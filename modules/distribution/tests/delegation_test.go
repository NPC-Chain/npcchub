package tests

import (
	"testing"

	"github.com/NPC-Chain/npcchub/modules/stake"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/stretchr/testify/require"
)

func TestWithdrawDelegationRewardBasic(t *testing.T) {
	ctx, accMapper, keeper, sk, fck := CreateTestInputAdvanced(t, false, sdk.NewIntWithDecimal(100, 18), sdk.ZeroDec())
	stakeHandler := stake.NewHandler(sk)
	denom := sk.BondDenom()

	//first make a validator
	msgCreateValidator := stake.NewTestMsgCreateValidator(valOpAddr1, valConsPk1, sdk.NewIntWithDecimal(10, 18))
	got := stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)
	_ = sk.ApplyAndReturnValidatorSetUpdates(ctx)

	// delegate
	msgDelegate := stake.NewTestMsgDelegate(delAddr1, valOpAddr1, sdk.NewIntWithDecimal(10, 18))
	got = stakeHandler(ctx, msgDelegate)
	require.True(t, got.IsOK())
	amt := accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewIntWithDecimal(90, 18), amt)

	// allocate 100 denom of fees
	feeInputs := sdk.NewIntWithDecimal(100, 18)
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)

	// withdraw delegation
	ctx = ctx.WithBlockHeight(1)
	sk.SetLastTotalPower(ctx, sdk.NewInt(10))
	sk.SetLastValidatorPower(ctx, valOpAddr1, sdk.NewInt(10))
	keeper.WithdrawDelegationReward(ctx, delAddr1, valOpAddr1)
	amt = accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)

	expRes := sdk.NewDecFromInt(sdk.NewIntWithDecimal(90, 18)).Add(sdk.NewDecFromInt(sdk.NewIntWithDecimal(100, 18)).Quo(sdk.NewDec(2))).TruncateInt() // 90 + 100 tokens * 10/20
	require.True(sdk.IntEq(t, expRes, amt))
}

func TestWithdrawDelegationRewardWithCommission(t *testing.T) {
	ctx, accMapper, keeper, sk, fck := CreateTestInputAdvanced(t, false, sdk.NewIntWithDecimal(100, 18), sdk.ZeroDec())
	stakeHandler := stake.NewHandler(sk)
	denom := sk.BondDenom()

	//first make a validator with 10% commission
	msgCreateValidator := stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr1, valConsPk1, sdk.NewIntWithDecimal(10, 18), sdk.NewDecWithPrec(1, 1))
	got := stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)
	_ = sk.ApplyAndReturnValidatorSetUpdates(ctx)

	// delegate
	msgDelegate := stake.NewTestMsgDelegate(delAddr1, valOpAddr1, sdk.NewIntWithDecimal(10, 18))
	got = stakeHandler(ctx, msgDelegate)
	require.True(t, got.IsOK())
	amt := accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewIntWithDecimal(90, 18), amt)

	// allocate 100 denom of fees
	feeInputs := sdk.NewIntWithDecimal(100, 18)
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)

	// withdraw delegation
	ctx = ctx.WithBlockHeight(1)
	keeper.WithdrawDelegationReward(ctx, delAddr1, valOpAddr1)
	amt = accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)

	expRes := sdk.NewDecFromInt(sdk.NewIntWithDecimal(90, 18)).Add(sdk.NewDecFromInt(sdk.NewIntWithDecimal(90, 18)).Quo(sdk.NewDec(2))).TruncateInt() // 90 + 100*90% tokens * 10/20
	require.True(sdk.IntEq(t, expRes, amt))
}

func TestWithdrawDelegationRewardTwoDelegators(t *testing.T) {
	ctx, accMapper, keeper, sk, fck := CreateTestInputAdvanced(t, false, sdk.NewIntWithDecimal(100, 18), sdk.ZeroDec())
	stakeHandler := stake.NewHandler(sk)
	denom := sk.BondDenom()

	//first make a validator with 10% commission
	msgCreateValidator := stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr1, valConsPk1, sdk.NewIntWithDecimal(10, 18), sdk.NewDecWithPrec(1, 1))
	got := stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)
	_ = sk.ApplyAndReturnValidatorSetUpdates(ctx)

	// delegate
	msgDelegate := stake.NewTestMsgDelegate(delAddr1, valOpAddr1, sdk.NewIntWithDecimal(10, 18))
	got = stakeHandler(ctx, msgDelegate)
	require.True(t, got.IsOK())
	amt := accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewIntWithDecimal(90, 18), amt)

	msgDelegate = stake.NewTestMsgDelegate(delAddr2, valOpAddr1, sdk.NewIntWithDecimal(20, 18))
	got = stakeHandler(ctx, msgDelegate)
	require.True(t, got.IsOK())
	amt = accMapper.GetAccount(ctx, delAddr2).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewIntWithDecimal(80, 18), amt)

	// allocate 100 denom of fees
	feeInputs := sdk.NewIntWithDecimal(100, 18)
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)

	// delegator 1 withdraw delegation
	ctx = ctx.WithBlockHeight(1)
	keeper.WithdrawDelegationReward(ctx, delAddr1, valOpAddr1)
	amt = accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)

	expRes := sdk.NewDecFromInt(sdk.NewIntWithDecimal(90, 18)).Add(sdk.NewDecFromInt(sdk.NewIntWithDecimal(90, 18)).Quo(sdk.NewDec(4))).TruncateInt() // 90 + 100*90% tokens * 10/40
	require.True(sdk.IntEq(t, expRes, amt))
}

// this test demonstrates how two delegators with the same power can end up
// with different rewards in the end
func TestWithdrawDelegationRewardTwoDelegatorsUneven(t *testing.T) {
	ctx, accMapper, keeper, sk, fck := CreateTestInputAdvanced(t, false, sdk.NewIntWithDecimal(100, 18), sdk.ZeroDec())
	stakeHandler := stake.NewHandler(sk)
	denom := sk.BondDenom()

	//first make a validator with no commission
	msgCreateValidator := stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr1, valConsPk1, sdk.NewIntWithDecimal(10, 18), sdk.ZeroDec())
	got := stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)
	_ = sk.ApplyAndReturnValidatorSetUpdates(ctx)

	// delegate
	msgDelegate := stake.NewTestMsgDelegate(delAddr1, valOpAddr1, sdk.NewIntWithDecimal(10, 18))
	got = stakeHandler(ctx, msgDelegate)
	require.True(t, got.IsOK())
	amt := accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewIntWithDecimal(90, 18), amt)

	msgDelegate = stake.NewTestMsgDelegate(delAddr2, valOpAddr1, sdk.NewIntWithDecimal(10, 18))
	got = stakeHandler(ctx, msgDelegate)
	require.True(t, got.IsOK())
	amt = accMapper.GetAccount(ctx, delAddr2).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewIntWithDecimal(90, 18), amt)

	// allocate 100 denom of fees
	feeInputs := sdk.NewIntWithDecimal(90, 18)
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)
	ctx = ctx.WithBlockHeight(1)

	// delegator 1 withdraw delegation early, delegator 2 just keeps it's accum
	keeper.WithdrawDelegationReward(ctx, delAddr1, valOpAddr1)
	amt = accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)

	expRes1 := sdk.NewDecFromInt(sdk.NewIntWithDecimal(90, 18)).Add(sdk.NewDecFromInt(sdk.NewIntWithDecimal(90, 18)).Quo(sdk.NewDec(3))).TruncateInt() // 90 + 100 * 10/30
	require.True(sdk.IntEq(t, expRes1, amt))

	// allocate 200 denom of fees
	feeInputs = sdk.NewIntWithDecimal(180, 18)
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)
	ctx = ctx.WithBlockHeight(2)

	// delegator 2 now withdraws everything it's entitled to
	keeper.WithdrawDelegationReward(ctx, delAddr2, valOpAddr1)
	amt = accMapper.GetAccount(ctx, delAddr2).GetCoins().AmountOf(denom)
	// existingTokens + (100+200 * (10/(20+30))
	withdrawnFromVal := sdk.NewDecFromInt(sdk.NewIntWithDecimal(60, 18).Add(sdk.NewIntWithDecimal(180, 18))).Mul(sdk.NewDec(2)).Quo(sdk.NewDec(5))
	expRes2 := sdk.NewDecFromInt(sdk.NewIntWithDecimal(90, 18)).Add(withdrawnFromVal).TruncateInt()
	require.True(sdk.IntEq(t, expRes2, amt))

	// finally delegator 1 withdraws the remainder of its reward
	keeper.WithdrawDelegationReward(ctx, delAddr1, valOpAddr1)
	amt = accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)

	remainingInVal := sdk.NewDecFromInt(sdk.NewIntWithDecimal(60, 18).Add(sdk.NewIntWithDecimal(180, 18))).Sub(withdrawnFromVal)
	expRes3 := sdk.NewDecFromInt(expRes1).Add(remainingInVal.Mul(sdk.NewDec(1)).Quo(sdk.NewDec(3))).TruncateInt()
	require.True(sdk.IntEq(t, expRes3, amt))

	// verify the final withdraw amounts are different
	require.True(t, expRes2.GT(expRes3))
}

func TestWithdrawDelegationRewardsAll(t *testing.T) {
	ctx, accMapper, keeper, sk, fck := CreateTestInputAdvanced(t, false, sdk.NewIntWithDecimal(100, 18), sdk.ZeroDec())
	stakeHandler := stake.NewHandler(sk)
	denom := sk.BondDenom()

	//make some  validators with different commissions
	msgCreateValidator := stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr1, valConsPk1, sdk.NewIntWithDecimal(10, 18), sdk.NewDecWithPrec(1, 1))
	got := stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)

	msgCreateValidator = stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr2, valConsPk2, sdk.NewIntWithDecimal(50, 18), sdk.NewDecWithPrec(2, 1))
	got = stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)

	msgCreateValidator = stake.NewTestMsgCreateValidatorWithCommission(
		valOpAddr3, valConsPk3, sdk.NewIntWithDecimal(40, 18), sdk.NewDecWithPrec(3, 1))
	got = stakeHandler(ctx, msgCreateValidator)
	require.True(t, got.IsOK(), "expected msg to be ok, got %v", got)

	// delegate to all the validators
	msgDelegate := stake.NewTestMsgDelegate(delAddr1, valOpAddr1, sdk.NewIntWithDecimal(10, 18))
	require.True(t, stakeHandler(ctx, msgDelegate).IsOK())
	msgDelegate = stake.NewTestMsgDelegate(delAddr1, valOpAddr2, sdk.NewIntWithDecimal(20, 18))
	require.True(t, stakeHandler(ctx, msgDelegate).IsOK())
	msgDelegate = stake.NewTestMsgDelegate(delAddr1, valOpAddr3, sdk.NewIntWithDecimal(30, 18))
	require.True(t, stakeHandler(ctx, msgDelegate).IsOK())

	// Update sk's LastValidatorPower/LastTotalPowers.
	_ = sk.ApplyAndReturnValidatorSetUpdates(ctx)

	// 40 tokens left after delegating 60 of them
	amt := accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)
	require.Equal(t, sdk.NewIntWithDecimal(40, 18), amt)

	// total power of each validator:
	// validator 1: 10 (self) + 10 (delegator) = 20
	// validator 2: 50 (self) + 20 (delegator) = 70
	// validator 3: 40 (self) + 30 (delegator) = 70
	// grand total: 160

	// allocate 100 denom of fees
	feeInputs := sdk.NewIntWithDecimal(1000, 18)
	fck.SetCollectedFees(sdk.Coins{sdk.NewCoin(denom, feeInputs)})
	require.Equal(t, feeInputs, fck.GetCollectedFees(ctx).AmountOf(denom))
	keeper.AllocateTokens(ctx, sdk.OneDec(), valConsAddr1)

	// withdraw delegation
	ctx = ctx.WithBlockHeight(1)
	keeper.WithdrawDelegationRewardsAll(ctx, delAddr1)
	amt = accMapper.GetAccount(ctx, delAddr1).GetCoins().AmountOf(denom)

	// orig-amount + fees *(1-proposerReward)* (val1Portion * delegatorPotion * (1-val1Commission) ... etc)
	//             + fees *(proposerReward)  * (delegatorPotion * (1-val1Commission))
	// 40          + 1000 *(1- 0.95)* (20/160 * 10/20 * 0.9 + 70/160 * 20/70 * 0.8 + 70/160 * 30/70 * 0.7)
	// 40          + 1000 *( 0.05)  * (10/20 * 0.9)
	feesInNonProposer := sdk.NewDecFromInt(feeInputs).Mul(sdk.NewDecWithPrec(95, 2))
	feesInProposer := sdk.NewDecFromInt(feeInputs).Mul(sdk.NewDecWithPrec(5, 2))
	feesInVal1 := feesInNonProposer.Mul(sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)).Quo(sdk.NewDecFromInt(sdk.NewIntWithDecimal(160, 18)))).Mul(sdk.NewDecWithPrec(9, 1))
	feesInVal2 := feesInNonProposer.Mul(sdk.NewDecFromInt(sdk.NewIntWithDecimal(20, 18)).Quo(sdk.NewDecFromInt(sdk.NewIntWithDecimal(160, 18)))).Mul(sdk.NewDecWithPrec(8, 1))
	feesInVal3 := feesInNonProposer.Mul(sdk.NewDecFromInt(sdk.NewIntWithDecimal(30, 18)).Quo(sdk.NewDecFromInt(sdk.NewIntWithDecimal(160, 18)))).Mul(sdk.NewDecWithPrec(7, 1))
	feesInVal1Proposer := feesInProposer.Mul(sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)).Quo(sdk.NewDecFromInt(sdk.NewIntWithDecimal(20, 18)))).Mul(sdk.NewDecWithPrec(9, 1))
	expRes := sdk.NewDecFromInt(sdk.NewIntWithDecimal(40, 18)).Add(feesInVal1).Add(feesInVal2).Add(feesInVal3).Add(feesInVal1Proposer).TruncateInt()
	require.True(sdk.IntEq(t, expRes, amt))
}
