package v0

import (
	"encoding/json"
	"fmt"

	"github.com/NPC-Chain/npcchub/app/protocol"
	"github.com/NPC-Chain/npcchub/codec"
	"github.com/NPC-Chain/npcchub/modules/auth"
	distr "github.com/NPC-Chain/npcchub/modules/distribution"
	"github.com/NPC-Chain/npcchub/modules/gov"
	"github.com/NPC-Chain/npcchub/modules/guardian"
	"github.com/NPC-Chain/npcchub/modules/mint"
	"github.com/NPC-Chain/npcchub/modules/service"
	"github.com/NPC-Chain/npcchub/modules/slashing"
	stake "github.com/NPC-Chain/npcchub/modules/stake"
	"github.com/NPC-Chain/npcchub/modules/upgrade"
	sdk "github.com/NPC-Chain/npcchub/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

// export the state of iris for a genesis file
func (p *ProtocolV0) ExportAppStateAndValidators(ctx sdk.Context, forZeroHeight bool) (
	appState json.RawMessage, validators []tmtypes.GenesisValidator, err error) {

	if forZeroHeight {
		p.prepForZeroHeightGenesis(ctx)
	}

	// iterate to get the accounts
	accounts := []GenesisAccount{}
	appendAccount := func(acc auth.Account) (stop bool) {
		account := NewGenesisAccountI(acc)
		accounts = append(accounts, account)
		return false
	}
	p.accountMapper.IterateAccounts(ctx, appendAccount)
	fileAccounts := []GenesisFileAccount{}
	for _, acc := range accounts {
		if acc.Coins == nil {
			continue
		}
		var coinsString []string
		for _, coin := range acc.Coins {
			coinsString = append(coinsString, coin.String())
		}
		fileAccounts = append(fileAccounts,
			GenesisFileAccount{
				Address:       acc.Address,
				Coins:         coinsString,
				Sequence:      acc.Sequence,
				AccountNumber: acc.AccountNumber,
			})
	}

	genState := NewGenesisFileState(
		fileAccounts,
		auth.ExportGenesis(ctx, p.feeKeeper),
		stake.ExportGenesis(ctx, p.StakeKeeper),
		mint.ExportGenesis(ctx, p.mintKeeper),
		distr.ExportGenesis(ctx, p.distrKeeper),
		gov.ExportGenesis(ctx, p.govKeeper),
		upgrade.ExportGenesis(ctx),
		service.ExportGenesis(ctx, p.serviceKeeper),
		guardian.ExportGenesis(ctx, p.guardianKeeper),
		slashing.ExportGenesis(ctx, p.slashingKeeper),
	)
	appState, err = codec.MarshalJSONIndent(p.cdc, genState)
	if err != nil {
		return nil, nil, err
	}

	validators = stake.WriteValidators(ctx, p.StakeKeeper)
	return appState, validators, nil
}

// prepare for fresh start at zero height
func (p *ProtocolV0) prepForZeroHeightGenesis(ctx sdk.Context) {

	/* Handle fee distribution state. */

	// withdraw all delegator & validator rewards
	vdiIter := func(_ int64, valInfo distr.ValidatorDistInfo) (stop bool) {
		_, _, err := p.distrKeeper.WithdrawValidatorRewardsAll(ctx, valInfo.OperatorAddr)
		if err != nil {
			panic(err)
		}
		return false
	}
	p.distrKeeper.IterateValidatorDistInfos(ctx, vdiIter)

	ddiIter := func(_ int64, distInfo distr.DelegationDistInfo) (stop bool) {
		_, err := p.distrKeeper.WithdrawDelegationReward(
			ctx, distInfo.DelegatorAddr, distInfo.ValOperatorAddr)
		if err != nil {
			panic(err)
		}
		return false
	}
	p.distrKeeper.IterateDelegationDistInfos(ctx, ddiIter)

	// set distribution info withdrawal heights to 0
	p.distrKeeper.IterateDelegationDistInfos(ctx, func(_ int64, delInfo distr.DelegationDistInfo) (stop bool) {
		delInfo.DelPoolWithdrawalHeight = 0
		p.distrKeeper.SetDelegationDistInfo(ctx, delInfo)
		return false
	})
	p.distrKeeper.IterateValidatorDistInfos(ctx, func(_ int64, valInfo distr.ValidatorDistInfo) (stop bool) {
		valInfo.FeePoolWithdrawalHeight = 0
		valInfo.DelAccum.UpdateHeight = 0
		p.distrKeeper.SetValidatorDistInfo(ctx, valInfo)
		return false
	})

	// assert that the fee pool is empty
	feePool := p.distrKeeper.GetFeePool(ctx)
	if !feePool.TotalValAccum.Accum.IsZero() {
		panic("unexpected leftover validator accum")
	}
	bondDenom := p.StakeKeeper.BondDenom()
	if !feePool.ValPool.AmountOf(bondDenom).IsZero() {
		panic(fmt.Sprintf("unexpected leftover validator pool coins: %v",
			feePool.ValPool.AmountOf(bondDenom).String()))
	}

	// reset fee pool height, save fee pool
	feePool.TotalValAccum = distr.NewTotalAccum(0)
	p.distrKeeper.SetFeePool(ctx, feePool)

	/* Handle stake state. */

	// iterate through redelegations, reset creation height
	p.StakeKeeper.IterateRedelegations(ctx, func(_ int64, red stake.Redelegation) (stop bool) {
		red.CreationHeight = 0
		p.StakeKeeper.SetRedelegation(ctx, red)
		return false
	})

	// iterate through unbonding delegations, reset creation height
	p.StakeKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd stake.UnbondingDelegation) (stop bool) {
		ubd.CreationHeight = 0
		p.StakeKeeper.SetUnbondingDelegation(ctx, ubd)
		return false
	})
	// Iterate through validators by power descending, reset bond and unbonding heights
	store := ctx.KVStore(protocol.KeyStake)
	iter := sdk.KVStoreReversePrefixIterator(store, stake.ValidatorsKey)
	defer iter.Close()
	counter := int16(0)
	var valConsAddrs []sdk.ConsAddress
	for ; iter.Valid(); iter.Next() {
		addr := sdk.ValAddress(iter.Key()[1:])
		validator, found := p.StakeKeeper.GetValidator(ctx, addr)
		if !found {
			panic("expected validator, not found")
		}
		validator.BondHeight = 0
		validator.UnbondingHeight = 0
		valConsAddrs = append(valConsAddrs, validator.ConsAddress())
		p.StakeKeeper.SetValidator(ctx, validator)
		counter++
	}

	/* Handle slashing state. */

	// remove all existing slashing periods and recreate one for each validator
	p.slashingKeeper.DeleteValidatorSlashingPeriods(ctx)
	for _, valConsAddr := range valConsAddrs {
		sp := slashing.ValidatorSlashingPeriod{
			ValidatorAddr: valConsAddr,
			StartHeight:   0,
			EndHeight:     0,
			SlashedSoFar:  sdk.ZeroDec(),
		}
		p.slashingKeeper.SetValidatorSlashingPeriod(ctx, sp)
	}

	// reset start height on signing infos
	p.slashingKeeper.IterateValidatorSigningInfos(ctx, func(addr sdk.ConsAddress, info slashing.ValidatorSigningInfo) (stop bool) {
		info.StartHeight = 0
		p.slashingKeeper.SetValidatorSigningInfo(ctx, addr, info)
		return false
	})

	/* Handle gov state. */

	gov.PrepForZeroHeightGenesis(ctx, p.govKeeper)

	/* Handle service state. */
	service.PrepForZeroHeightGenesis(ctx, p.serviceKeeper)
}
