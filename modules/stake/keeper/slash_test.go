package keeper

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/NPC-Chain/npcchub/modules/stake/types"
	sdk "github.com/NPC-Chain/npcchub/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// TODO integrate with test_common.go helper (CreateTestInput)
// setup helper function - creates two validators
func setupHelper(t *testing.T, amt sdk.Int) (sdk.Context, Keeper, types.Params) {

	// setup
	ctx, _, keeper := CreateTestInput(t, false, amt)
	params := keeper.GetParams(ctx)
	pool := keeper.GetPool(ctx)
	numVals := 3
	pool.BankKeeper.IncreaseLoosenToken(ctx, sdk.Coins{sdk.NewCoin(types.StakeDenom, amt.Mul(sdk.NewInt(int64(numVals))))})

	// add numVals validators
	for i := 0; i < numVals; i++ {
		validator := types.NewValidator(addrVals[i], PKs[i], types.Description{})
		validator, pool, _ = validator.AddTokensFromDel(ctx, pool, amt)
		pool.BondedPool.BondedTokens = pool.BondedPool.BondedTokens.Add(sdk.NewDecFromInt(amt))
		keeper.SetPool(ctx, pool)
		validator = TestingUpdateValidator(keeper, ctx, validator, true)
		keeper.SetValidatorByConsAddr(ctx, validator)
	}
	pool = keeper.GetPool(ctx)

	return ctx, keeper, params
}

//_________________________________________________________________________________

// tests Jail, Unjail
func TestRevocation(t *testing.T) {

	// setup
	ctx, keeper, _ := setupHelper(t, sdk.NewIntWithDecimal(10, 18))
	addr := addrVals[0]
	consAddr := sdk.ConsAddress(PKs[0].Address())

	// initial state
	val, found := keeper.GetValidator(ctx, addr)
	require.True(t, found)
	require.False(t, val.GetJailed())

	// test jail
	keeper.Jail(ctx, consAddr)
	val, found = keeper.GetValidator(ctx, addr)
	require.True(t, found)
	require.True(t, val.GetJailed())

	// test unjail
	keeper.Unjail(ctx, consAddr)
	val, found = keeper.GetValidator(ctx, addr)
	require.True(t, found)
	require.False(t, val.GetJailed())
}

// tests slashUnbondingDelegation
func TestSlashUnbondingDelegation(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, sdk.NewIntWithDecimal(10, 18))
	fraction := sdk.NewDecWithPrec(5, 1)

	// set an unbonding delegation
	ubd := types.UnbondingDelegation{
		DelegatorAddr:  addrDels[0],
		ValidatorAddr:  addrVals[0],
		CreationHeight: 0,
		// expiration timestamp (beyond which the unbonding delegation shouldn't be slashed)
		MinTime:        time.Unix(0, 0),
		InitialBalance: sdk.NewInt64Coin(keeper.BondDenom(), 10),
		Balance:        sdk.NewInt64Coin(keeper.BondDenom(), 10),
	}
	keeper.SetUnbondingDelegation(ctx, ubd)

	// unbonding started prior to the infraction height, stake didn't contribute
	slashAmount, _ := keeper.slashUnbondingDelegation(ctx, ubd, 1, fraction)
	require.Equal(t, int64(0), slashAmount.RoundInt64())

	// after the expiration time, no longer eligible for slashing
	ctx = ctx.WithBlockHeader(abci.Header{Time: time.Unix(10, 0)})
	keeper.SetUnbondingDelegation(ctx, ubd)
	slashAmount, _ = keeper.slashUnbondingDelegation(ctx, ubd, 0, fraction)
	require.Equal(t, int64(0), slashAmount.RoundInt64())

	// test valid slash, before expiration timestamp and to which stake contributed
	oldPool := keeper.GetPool(ctx)
	oldPoolLoosenToken := oldPool.GetLoosenTokenAmount(ctx)
	ctx = ctx.WithBlockHeader(abci.Header{Time: time.Unix(0, 0)})
	keeper.SetUnbondingDelegation(ctx, ubd)
	slashAmount, _ = keeper.slashUnbondingDelegation(ctx, ubd, 0, fraction)
	require.Equal(t, int64(5), slashAmount.RoundInt64())
	ubd, found := keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)

	// initialbalance unchanged
	require.Equal(t, sdk.NewInt64Coin(keeper.BondDenom(), 10), ubd.InitialBalance)

	// balance decreased
	require.Equal(t, sdk.NewInt64Coin(keeper.BondDenom(), 5), ubd.Balance)
	newPool := keeper.GetPool(ctx)
	require.Equal(t, int64(5), oldPoolLoosenToken.Sub(newPool.GetLoosenTokenAmount(ctx)).RoundInt64())
}

// tests slashRedelegation
func TestSlashRedelegation(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, sdk.NewIntWithDecimal(10, 18))
	fraction := sdk.NewDecWithPrec(5, 1)

	// set a redelegation
	rd := types.Redelegation{
		DelegatorAddr:    addrDels[0],
		ValidatorSrcAddr: addrVals[0],
		ValidatorDstAddr: addrVals[1],
		CreationHeight:   0,
		// expiration timestamp (beyond which the redelegation shouldn't be slashed)
		MinTime:        time.Unix(0, 0),
		SharesSrc:      sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)),
		SharesDst:      sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)),
		InitialBalance: sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(10, 18)),
		Balance:        sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(10, 18)),
	}
	keeper.SetRedelegation(ctx, rd)

	// set the associated delegation
	del := types.Delegation{
		DelegatorAddr: addrDels[0],
		ValidatorAddr: addrVals[1],
		Shares:        sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)),
	}
	keeper.SetDelegation(ctx, del)

	// started redelegating prior to the current height, stake didn't contribute to infraction
	validator, found := keeper.GetValidator(ctx, addrVals[1])
	require.True(t, found)
	slashAmount, _ := keeper.slashRedelegation(ctx, validator, rd, 1, fraction)
	require.Equal(t, sdk.ZeroDec(), slashAmount)

	// after the expiration time, no longer eligible for slashing
	ctx = ctx.WithBlockHeader(abci.Header{Time: time.Unix(10, 0)})
	keeper.SetRedelegation(ctx, rd)
	validator, found = keeper.GetValidator(ctx, addrVals[1])
	require.True(t, found)
	slashAmount, _ = keeper.slashRedelegation(ctx, validator, rd, 0, fraction)
	require.Equal(t, sdk.ZeroDec(), slashAmount)

	// test valid slash, before expiration timestamp and to which stake contributed
	oldPool := keeper.GetPool(ctx)
	ctx = ctx.WithBlockHeader(abci.Header{Time: time.Unix(0, 0)})
	keeper.SetRedelegation(ctx, rd)
	validator, found = keeper.GetValidator(ctx, addrVals[1])
	require.True(t, found)
	slashAmount, _ = keeper.slashRedelegation(ctx, validator, rd, 0, fraction)
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(5, 18)), slashAmount)
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)

	// end block
	updates := keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.Equal(t, 1, len(updates))

	// initialbalance unchanged
	require.Equal(t, sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(10, 18)), rd.InitialBalance)

	// balance decreased
	require.Equal(t, sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(5, 18)), rd.Balance)

	// shares decreased
	del, found = keeper.GetDelegation(ctx, addrDels[0], addrVals[1])
	require.True(t, found)
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(5, 18)), del.Shares)

	// pool bonded tokens decreased
	newPool := keeper.GetPool(ctx)
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(5, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
}

// tests Slash at a future height (must panic)
func TestSlashAtFutureHeight(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, sdk.NewIntWithDecimal(10, 18))
	consAddr := sdk.ConsAddress(PKs[0].Address())
	fraction := sdk.NewDecWithPrec(5, 1)
	require.Panics(t, func() { keeper.Slash(ctx, consAddr, 1, 10, fraction) })
}

// test slash at a negative height
// this just represents pre-genesis and should have the same effect as slashing at height 0
func TestSlashAtNegativeHeight(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, sdk.NewIntWithDecimal(10, 18))
	consAddr := sdk.ConsAddress(PKs[0].Address())
	fraction := sdk.NewDecWithPrec(5, 1)

	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	keeper.Slash(ctx, consAddr, -2, 10, fraction)

	// read updated state
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	newPool := keeper.GetPool(ctx)

	// end block
	updates := keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.Equal(t, 1, len(updates), "cons addr: %v, updates: %v", []byte(consAddr), updates)

	validator = keeper.mustGetValidator(ctx, validator.OperatorAddr)
	// power decreased
	require.Equal(t, sdk.NewDec(5), validator.GetPower())
	// pool bonded shares decreased
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(5, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
}

// tests Slash at the current height
func TestSlashValidatorAtCurrentHeight(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, sdk.NewIntWithDecimal(10, 18))
	consAddr := sdk.ConsAddress(PKs[0].Address())
	fraction := sdk.NewDecWithPrec(5, 1)

	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	keeper.Slash(ctx, consAddr, ctx.BlockHeight(), 10, fraction)

	// read updated state
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	newPool := keeper.GetPool(ctx)

	// end block
	updates := keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.Equal(t, 1, len(updates), "cons addr: %v, updates: %v", []byte(consAddr), updates)

	validator = keeper.mustGetValidator(ctx, validator.OperatorAddr)
	// power decreased
	require.Equal(t, sdk.NewDec(5), validator.GetPower())
	// pool bonded shares decreased
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(5, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
}

// tests Slash at a previous height with an unbonding delegation
func TestSlashWithUnbondingDelegation(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, sdk.NewIntWithDecimal(10, 18))
	consAddr := sdk.ConsAddress(PKs[0].Address())
	fraction := sdk.NewDecWithPrec(5, 1)

	// set an unbonding delegation
	ubd := types.UnbondingDelegation{
		DelegatorAddr:  addrDels[0],
		ValidatorAddr:  addrVals[0],
		CreationHeight: 11,
		// expiration timestamp (beyond which the unbonding delegation shouldn't be slashed)
		MinTime:        time.Unix(0, 0),
		InitialBalance: sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(4, 18)),
		Balance:        sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(4, 18)),
	}
	keeper.SetUnbondingDelegation(ctx, ubd)

	// slash validator for the first time
	ctx = ctx.WithBlockHeight(12)
	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	keeper.Slash(ctx, consAddr, 10, 10, fraction)

	// end block
	updates := keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.Equal(t, 1, len(updates))

	// read updating unbonding delegation
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance decreased
	require.Equal(t, sdk.NewIntWithDecimal(2, 18), ubd.Balance.Amount)
	// read updated pool
	newPool := keeper.GetPool(ctx)
	// bonded tokens burned
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(3, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	// power decreased by 3 - 6 stake originally bonded at the time of infraction
	// was still bonded at the time of discovery and was slashed by half, 4 stake
	// bonded at the time of discovery hadn't been bonded at the time of infraction
	// and wasn't slashed
	require.Equal(t, sdk.NewDec(7), validator.GetPower())

	// slash validator again
	ctx = ctx.WithBlockHeight(13)
	keeper.Slash(ctx, consAddr, 9, 10, fraction)
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance decreased again
	require.Equal(t, sdk.NewInt(0), ubd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// bonded tokens burned again
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(6, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	// power decreased by 3 again
	require.Equal(t, sdk.NewDec(4), validator.GetPower())

	// slash validator again
	// all originally bonded stake has been slashed, so this will have no effect
	// on the unbonding delegation, but it will slash stake bonded since the infraction
	// this may not be the desirable behaviour, ref https://github.com/NPC-Chain/npcchub/issues/1440
	ctx = ctx.WithBlockHeight(13)
	keeper.Slash(ctx, consAddr, 9, 10, fraction)
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance unchanged
	require.Equal(t, sdk.NewInt(0), ubd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// bonded tokens burned again
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(9, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	// power decreased by 3 again
	require.Equal(t, sdk.NewDec(1), validator.GetPower())

	// slash validator again
	// all originally bonded stake has been slashed, so this will have no effect
	// on the unbonding delegation, but it will slash stake bonded since the infraction
	// this may not be the desirable behaviour, ref https://github.com/NPC-Chain/npcchub/issues/1440
	ctx = ctx.WithBlockHeight(13)
	keeper.Slash(ctx, consAddr, 9, 10, fraction)
	ubd, found = keeper.GetUnbondingDelegation(ctx, addrDels[0], addrVals[0])
	require.True(t, found)
	// balance unchanged
	require.Equal(t, sdk.NewInt(0), ubd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// just 1 bonded token burned again since that's all the validator now has
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
	// apply TM updates
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	// read updated validator
	// power decreased by 1 again, validator is out of stake
	// validator should be in unbonding period
	validator, _ = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.Equal(t, validator.GetStatus(), sdk.Unbonding)
}

// tests Slash at a previous height with a redelegation
func TestSlashWithRedelegation(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, sdk.NewIntWithDecimal(10, 18))
	consAddr := sdk.ConsAddress(PKs[0].Address())
	fraction := sdk.NewDecWithPrec(5, 1)

	// set a redelegation
	rd := types.Redelegation{
		DelegatorAddr:    addrDels[0],
		ValidatorSrcAddr: addrVals[0],
		ValidatorDstAddr: addrVals[1],
		CreationHeight:   11,
		MinTime:          time.Unix(0, 0),
		SharesSrc:        sdk.NewDecFromInt(sdk.NewIntWithDecimal(6, 18)),
		SharesDst:        sdk.NewDecFromInt(sdk.NewIntWithDecimal(6, 18)),
		InitialBalance:   sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(6, 18)),
		Balance:          sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(6, 18)),
	}
	keeper.SetRedelegation(ctx, rd)

	// set the associated delegation
	del := types.Delegation{
		DelegatorAddr: addrDels[0],
		ValidatorAddr: addrVals[1],
		Shares:        sdk.NewDecFromInt(sdk.NewIntWithDecimal(6, 18)),
	}
	keeper.SetDelegation(ctx, del)

	// update bonded tokens
	pool := keeper.GetPool(ctx)
	pool.BondedPool.BondedTokens = pool.BondedPool.BondedTokens.Add(sdk.NewDecFromInt(sdk.NewIntWithDecimal(6, 18)))
	keeper.SetPool(ctx, pool)

	// slash validator
	ctx = ctx.WithBlockHeight(12)
	oldPool := keeper.GetPool(ctx)
	validator, found := keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	keeper.Slash(ctx, consAddr, 10, 10, fraction)

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance decreased
	require.Equal(t, sdk.NewIntWithDecimal(3, 18), rd.Balance.Amount)
	// read updated pool
	newPool := keeper.GetPool(ctx)
	// bonded tokens burned
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(5, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	// power decreased by 2 - 4 stake originally bonded at the time of infraction
	// was still bonded at the time of discovery and was slashed by half, 4 stake
	// bonded at the time of discovery hadn't been bonded at the time of infraction
	// and wasn't slashed
	require.Equal(t, sdk.NewDec(8), validator.GetPower())

	// slash the validator again
	ctx = ctx.WithBlockHeight(12)
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	require.NotPanics(t, func() { keeper.Slash(ctx, consAddr, 10, 10, sdk.OneDec()) })

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance decreased, now zero
	require.Equal(t, sdk.NewInt(0), rd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// seven bonded tokens burned
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(12, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	// power decreased by 4
	require.Equal(t, sdk.NewDec(4), validator.GetPower())

	// slash the validator again, by 100%
	ctx = ctx.WithBlockHeight(12)
	validator, found = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.True(t, found)
	keeper.Slash(ctx, consAddr, 10, 10, sdk.OneDec())

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance still zero
	require.Equal(t, sdk.NewInt(0), rd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// four more bonded tokens burned
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(16, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
	// apply TM updates
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)
	// read updated validator
	// validator decreased to zero power, should be in unbonding period
	validator, _ = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.Equal(t, validator.GetStatus(), sdk.Unbonding)

	// slash the validator again, by 100%
	// no stake remains to be slashed
	ctx = ctx.WithBlockHeight(12)
	// validator still in unbonding period
	validator, _ = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.Equal(t, validator.GetStatus(), sdk.Unbonding)
	keeper.Slash(ctx, consAddr, 10, 10, sdk.OneDec())

	// read updating redelegation
	rd, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance still zero
	require.Equal(t, sdk.NewInt(0), rd.Balance.Amount)
	// read updated pool
	newPool = keeper.GetPool(ctx)
	// no more bonded tokens burned
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(16, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
	// read updated validator
	// power still zero, still in unbonding period
	validator, _ = keeper.GetValidatorByConsAddr(ctx, consAddr)
	require.Equal(t, validator.GetStatus(), sdk.Unbonding)
}

// tests Slash at a previous height with both an unbonding delegation and a redelegation
func TestSlashBoth(t *testing.T) {
	ctx, keeper, _ := setupHelper(t, sdk.NewIntWithDecimal(10, 18))
	fraction := sdk.NewDecWithPrec(5, 1)

	// set a redelegation
	rdA := types.Redelegation{
		DelegatorAddr:    addrDels[0],
		ValidatorSrcAddr: addrVals[0],
		ValidatorDstAddr: addrVals[1],
		CreationHeight:   11,
		// expiration timestamp (beyond which the redelegation shouldn't be slashed)
		MinTime:        time.Unix(0, 0),
		SharesSrc:      sdk.NewDecFromInt(sdk.NewIntWithDecimal(6, 18)),
		SharesDst:      sdk.NewDecFromInt(sdk.NewIntWithDecimal(6, 18)),
		InitialBalance: sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(6, 18)),
		Balance:        sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(6, 18)),
	}
	keeper.SetRedelegation(ctx, rdA)

	// set the associated delegation
	delA := types.Delegation{
		DelegatorAddr: addrDels[0],
		ValidatorAddr: addrVals[1],
		Shares:        sdk.NewDecFromInt(sdk.NewIntWithDecimal(6, 18)),
	}
	keeper.SetDelegation(ctx, delA)

	// set an unbonding delegation
	ubdA := types.UnbondingDelegation{
		DelegatorAddr:  addrDels[0],
		ValidatorAddr:  addrVals[0],
		CreationHeight: 11,
		// expiration timestamp (beyond which the unbonding delegation shouldn't be slashed)
		MinTime:        time.Unix(0, 0),
		InitialBalance: sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(4, 18)),
		Balance:        sdk.NewCoin(keeper.BondDenom(), sdk.NewIntWithDecimal(4, 18)),
	}
	keeper.SetUnbondingDelegation(ctx, ubdA)

	// slash validator
	ctx = ctx.WithBlockHeight(12)
	oldPool := keeper.GetPool(ctx)
	oldPoolLoosenToken := oldPool.GetLoosenTokenAmount(ctx)
	validator, found := keeper.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(PKs[0]))
	require.True(t, found)
	consAddr0 := sdk.ConsAddress(PKs[0].Address())
	keeper.Slash(ctx, consAddr0, 10, 10, fraction)

	// read updating redelegation
	rdA, found = keeper.GetRedelegation(ctx, addrDels[0], addrVals[0], addrVals[1])
	require.True(t, found)
	// balance decreased
	require.Equal(t, sdk.NewIntWithDecimal(3, 18), rdA.Balance.Amount)
	// read updated pool
	newPool := keeper.GetPool(ctx)
	// loose tokens burned
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(2, 18)), oldPoolLoosenToken.Sub(newPool.GetLoosenTokenAmount(ctx)))
	// bonded tokens burned
	require.Equal(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(3, 18)), oldPool.BondedPool.BondedTokens.Sub(newPool.BondedPool.BondedTokens))
	// read updated validator
	validator, found = keeper.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(PKs[0]))
	require.True(t, found)
	// power not decreased, all stake was bonded since
	require.Equal(t, sdk.NewDec(10), validator.GetPower())
}
