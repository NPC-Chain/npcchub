package slashing

import (
	"fmt"
	"github.com/NPC-Chain/npcchub/codec"
	"github.com/NPC-Chain/npcchub/modules/params"
	stake "github.com/NPC-Chain/npcchub/modules/stake/types"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/tendermint/tendermint/crypto"
)

// Keeper of the slashing store
type Keeper struct {
	storeKey     sdk.StoreKey
	cdc          *codec.Codec
	validatorSet sdk.ValidatorSet
	paramspace   params.Subspace

	// codespace
	codespace sdk.CodespaceType
	// metrics
	metrics *Metrics
}

// NewKeeper creates a slashing keeper
func NewKeeper(cdc *codec.Codec, key sdk.StoreKey, vs sdk.ValidatorSet, paramspace params.Subspace, codespace sdk.CodespaceType, metrics *Metrics) Keeper {
	keeper := Keeper{
		storeKey:     key,
		cdc:          cdc,
		validatorSet: vs,
		paramspace:   paramspace.WithTypeTable(ParamTypeTable()),
		codespace:    codespace,
		metrics:      metrics,
	}
	return keeper
}

// handle a validator signing two blocks at the same height
// power: power of the double-signing validator at the height of infraction
func (k Keeper) handleDoubleSign(ctx sdk.Context, addr crypto.Address, infractionHeight int64, power int64) (tags sdk.Tags) {
	logger := ctx.Logger()
	time := ctx.BlockHeader().Time
	age := ctx.BlockHeight() - infractionHeight
	consAddr := sdk.ConsAddress(addr)

	// To resolve https://github.com/NPC-Chain/npcchub/issues/1334
	// Unbonding period is calculated by time however evidence age is calculated by height.
	// It is possible that the unbonding period is completed, but evidence is still valid
	// If a validator is removed once unbonding period is completed, then its pubkey will be deleted, which will result in panic and consensus halt
	// So here we check the validator existence and validator status first
	validator := k.validatorSet.ValidatorByConsAddr(ctx, consAddr)
	if validator == nil || validator.GetStatus() == sdk.Unbonded {
		return
	}

	pubkey, err := k.getPubkey(ctx, addr)
	if err != nil {
		panic(fmt.Sprintf("Validator consensus-address %v not found", consAddr))
	}

	// Double sign too old
	maxEvidenceAge := k.MaxEvidenceAge(ctx)
	if age > maxEvidenceAge {
		logger.Info("Ignored double sign because of the age is greater than max age", "validator", pubkey.Address(),
			"infraction_height", infractionHeight, "age", age, "max_evidence_age", maxEvidenceAge)
		return
	}

	// Double sign confirmed
	logger.Info("Validator double sign Confirmed", "validator", pubkey.Address(), "infraction_height", infractionHeight,
		"age", age, "max_evidence_age", maxEvidenceAge)

	// We need to retrieve the stake distribution which signed the block, so we subtract ValidatorUpdateDelay from the evidence height.
	// Note that this *can* result in a negative "distributionHeight", up to -ValidatorUpdateDelay,
	// i.e. at the end of the pre-genesis block (none) = at the beginning of the genesis block.
	// That's fine since this is just used to filter unbonding delegations & redelegations.
	distributionHeight := infractionHeight - stake.ValidatorUpdateDelay

	// Cap the amount slashed to the penalty for the worst infraction
	// within the slashing period when this infraction was committed
	fraction := k.SlashFractionDoubleSign(ctx)
	revisedFraction := k.capBySlashingPeriod(ctx, consAddr, fraction, distributionHeight)
	logger.Info("Fraction slashed capped by slashing period", "fraction", fraction, "revised_fraction", revisedFraction)

	// Slash validator
	// `power` is the int64 power of the validator as provided to/by
	// Tendermint. This value is validator.Tokens as sent to Tendermint via
	// ABCI, and now received as evidence.
	// The revisedFraction (which is the new fraction to be slashed) is passed
	// in separately to separately slash unbonding and rebonding delegations.
	tags = k.validatorSet.Slash(ctx, consAddr, distributionHeight, power, revisedFraction)

	// Jail validator if not already jailed
	if !validator.GetJailed() {
		k.validatorSet.Jail(ctx, consAddr)
	}

	// Set or updated validator jail duration
	signInfo, found := k.getValidatorSigningInfo(ctx, consAddr)
	if !found {
		panic(fmt.Sprintf("Expected signing info for validator %s but not found", consAddr))
	}
	signInfo.JailedUntil = time.Add(k.DoubleSignJailDuration(ctx))
	k.SetValidatorSigningInfo(ctx, consAddr, signInfo)
	return
}

// handle a validator signature, must be called once per validator per block
// TODO refactor to take in a consensus address, additionally should maybe just take in the pubkey too
func (k Keeper) handleValidatorSignature(ctx sdk.Context, addr crypto.Address, power int64, signed bool) (tags sdk.Tags) {
	logger := ctx.Logger()
	height := ctx.BlockHeight()
	consAddr := sdk.ConsAddress(addr)
	pubkey, err := k.getPubkey(ctx, addr)
	if err != nil {
		panic(fmt.Sprintf("Validator consensus-address %v not found", consAddr))
	}
	// Local index, so counts blocks validator *should* have signed
	// Will use the 0-value default signing info if not present, except for start height
	signInfo, found := k.getValidatorSigningInfo(ctx, consAddr)
	if !found {
		panic(fmt.Sprintf("Expected signing info for validator %s but not found", consAddr))
	}
	index := signInfo.IndexOffset % k.SignedBlocksWindow(ctx)
	signInfo.IndexOffset++

	// Update signed block bit array & counter
	// This counter just tracks the sum of the bit array
	// That way we avoid needing to read/write the whole array each time
	previous := k.getValidatorMissedBlockBitArray(ctx, consAddr, index)
	missed := !signed
	switch {
	case !previous && missed:
		// Array value has changed from not missed to missed, increment counter
		k.setValidatorMissedBlockBitArray(ctx, consAddr, index, true)
		signInfo.MissedBlocksCounter++
	case previous && !missed:
		// Array value has changed from missed to not missed, decrement counter
		k.setValidatorMissedBlockBitArray(ctx, consAddr, index, false)
		signInfo.MissedBlocksCounter--
	default:
		// Array value at this index has not changed, no need to update counter
	}

	if missed {
		logger.Info("Absent validator", "validator", addr.String(), "missed", signInfo.MissedBlocksCounter, "threshold", k.MinSignedPerWindow(ctx))
	}
	minHeight := signInfo.StartHeight + k.SignedBlocksWindow(ctx)
	maxMissed := k.SignedBlocksWindow(ctx) - k.MinSignedPerWindow(ctx)
	if height > minHeight && signInfo.MissedBlocksCounter > maxMissed {
		validator := k.validatorSet.ValidatorByConsAddr(ctx, consAddr)
		if validator != nil && !validator.GetJailed() {
			// Downtime confirmed: slash and jail the validator
			logger.Info("Validator Downtime confirmed", "validator", pubkey.Address(),
				"operator_address", validator.GetOperator().String(), "min_height", minHeight, "missed", signInfo.MissedBlocksCounter, "threshold", k.MinSignedPerWindow(ctx))
			// We need to retrieve the stake distribution which signed the block, so we subtract ValidatorUpdateDelay from the evidence height,
			// and subtract an additional 1 since this is the LastCommit.
			// Note that this *can* result in a negative "distributionHeight" up to -ValidatorUpdateDelay-1,
			// i.e. at the end of the pre-genesis block (none) = at the beginning of the genesis block.
			// That's fine since this is just used to filter unbonding delegations & redelegations.
			distributionHeight := height - stake.ValidatorUpdateDelay - 1
			slashTags := k.validatorSet.Slash(ctx, consAddr, distributionHeight, power, k.SlashFractionDowntime(ctx))
			tags = tags.AppendTags(slashTags)
			k.validatorSet.Jail(ctx, consAddr)
			signInfo.JailedUntil = ctx.BlockHeader().Time.Add(k.DowntimeJailDuration(ctx))
			// We need to reset the counter & array so that the validator won't be immediately slashed for downtime upon rebonding.
			signInfo.MissedBlocksCounter = 0
			signInfo.IndexOffset = 0
			k.clearValidatorMissedBlockBitArray(ctx, consAddr)
		} else {
			// Validator was (a) not found or (b) already jailed, don't slash
			logger.Info("Validator would have been slashed for downtime, but was either not found in store or already jailed",
				"validator", pubkey.Address())
		}
	}

	// Set the updated signing info
	k.SetValidatorSigningInfo(ctx, consAddr, signInfo)
	return
}

// Punish proposer censorship by slashing malefactor's stake
func (k Keeper) handleProposerCensorship(ctx sdk.Context, addr crypto.Address, infractionHeight int64) (tags sdk.Tags) {
	logger := ctx.Logger()
	time := ctx.BlockHeader().Time
	consAddr := sdk.ConsAddress(addr)
	_, err := k.getPubkey(ctx, addr)
	if err != nil {
		panic(fmt.Sprintf("Validator consensus-address %v not found", consAddr))
	}

	// Get validator.
	validator := k.validatorSet.ValidatorByConsAddr(ctx, consAddr)
	if validator == nil || validator.GetStatus() == sdk.Unbonded {
		// Defensive.
		// Simulation doesn't take unbonding periods into account, and
		// Tendermint might break this assumption at some point.
		return
	}
	logger.Info("The malefactor proposer proposed a invalid block",
		"proposer_address", validator.GetOperator().String(),
		"block_height", ctx.BlockHeight(), "consensus_address", consAddr.String())

	distributionHeight := infractionHeight - stake.ValidatorUpdateDelay
	// Slash validator
	// `power` is the int64 power of the validator as provided to/by
	// Tendermint. This value is validator.Tokens as sent to Tendermint via
	// ABCI, and now received as evidence.
	// The revisedFraction (which is the new fraction to be slashed) is passed
	// in separately to separately slash unbonding and rebonding delegations.
	tags = k.validatorSet.Slash(ctx, consAddr, distributionHeight, validator.GetPower().RoundInt64(), k.SlashFractionCensorship(ctx))

	// Jail validator if not already jailed
	if !validator.GetJailed() {
		k.validatorSet.Jail(ctx, consAddr)
	}

	// Set or updated validator jail duration
	signInfo, found := k.getValidatorSigningInfo(ctx, consAddr)
	if !found {
		panic(fmt.Sprintf("Expected signing info for validator %s but not found", consAddr))
	}
	signInfo.JailedUntil = time.Add(k.CensorshipJailDuration(ctx))
	k.SetValidatorSigningInfo(ctx, consAddr, signInfo)
	return
}

func (k Keeper) addPubkey(ctx sdk.Context, pubkey crypto.PubKey) {
	addr := pubkey.Address()
	k.setAddrPubkeyRelation(ctx, addr, pubkey)
}

func (k Keeper) getPubkey(ctx sdk.Context, address crypto.Address) (crypto.PubKey, error) {
	store := ctx.KVStore(k.storeKey)
	var pubkey crypto.PubKey
	err := k.cdc.UnmarshalBinaryLengthPrefixed(store.Get(getAddrPubkeyRelationKey(address)), &pubkey)
	if err != nil {
		return nil, fmt.Errorf("address %v not found", address)
	}
	return pubkey, nil
}

func (k Keeper) setAddrPubkeyRelation(ctx sdk.Context, addr crypto.Address, pubkey crypto.PubKey) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshalBinaryLengthPrefixed(pubkey)
	store.Set(getAddrPubkeyRelationKey(addr), bz)
}

func (k Keeper) deleteAddrPubkeyRelation(ctx sdk.Context, addr crypto.Address) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(getAddrPubkeyRelationKey(addr))
}
