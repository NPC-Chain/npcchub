package stake

import (
	"fmt"
	"github.com/NPC-Chain/npcchub/mock"
	"github.com/NPC-Chain/npcchub/mock/baseapp"
	"github.com/NPC-Chain/npcchub/mock/simulation"
	"github.com/NPC-Chain/npcchub/modules/auth"
	"github.com/NPC-Chain/npcchub/modules/stake"
	"github.com/NPC-Chain/npcchub/modules/stake/keeper"
	"github.com/NPC-Chain/npcchub/modules/stake/types"
	sdk "github.com/NPC-Chain/npcchub/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"math/rand"
)

// SimulateMsgCreateValidator
func SimulateMsgCreateValidator(m auth.AccountKeeper, k stake.Keeper) simulation.Operation {
	handler := stake.NewHandler(k)
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simulation.Account, event func(string)) (
		action string, fOp []simulation.FutureOperation, err error) {

		denom := k.BondDenom()
		description := stake.Description{
			Moniker: simulation.RandStringOfLength(r, 10),
		}

		maxCommission := sdk.NewInt(10)
		commission := stake.NewCommissionMsg(
			sdk.NewDecWithPrec(simulation.RandomAmount(r, maxCommission).Int64(), 1),
			sdk.NewDecWithPrec(simulation.RandomAmount(r, maxCommission).Int64(), 1),
			sdk.NewDecWithPrec(simulation.RandomAmount(r, maxCommission).Int64(), 1),
		)

		acc := simulation.RandomAcc(r, accs)
		address := sdk.ValAddress(acc.Address)

		amount := m.GetAccount(ctx, acc.Address).GetCoins().AmountOf(denom)

		if amount.GT(sdk.ZeroInt()) {
			amount = simulation.RandomAmount(r, amount)
		}

		if amount.Equal(sdk.ZeroInt()) {
			return "no-operation", nil, nil
		}

		msg := stake.MsgCreateValidator{
			Description:   description,
			Commission:    commission,
			ValidatorAddr: address,
			DelegatorAddr: acc.Address,
			PubKey:        acc.PubKey,
			Delegation:    sdk.NewCoin(denom, amount),
		}

		if msg.ValidateBasic() != nil {
			return "", nil, fmt.Errorf("expected msg to pass ValidateBasic: %s", msg.GetSignBytes())
		}

		ctx, write := ctx.CacheContext()
		result := handler(ctx, msg)
		if result.IsOK() {
			write()
		}

		event(fmt.Sprintf("stake/MsgCreateValidator/%v", result.IsOK()))

		// require.True(t, result.IsOK(), "expected OK result but instead got %v", result)
		action = fmt.Sprintf("TestMsgCreateValidator: ok %v, msg %s", result.IsOK(), msg.GetSignBytes())
		return action, nil, nil
	}
}

// SimulateMsgEditValidator
func SimulateMsgEditValidator(k stake.Keeper) simulation.Operation {
	handler := stake.NewHandler(k)
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simulation.Account, event func(string)) (
		action string, fOp []simulation.FutureOperation, err error) {

		description := stake.Description{
			Moniker:  simulation.RandStringOfLength(r, 10),
			Identity: simulation.RandStringOfLength(r, 10),
			Website:  simulation.RandStringOfLength(r, 10),
			Details:  simulation.RandStringOfLength(r, 10),
		}

		maxCommission := sdk.NewInt(10)
		newCommissionRate := sdk.NewDecWithPrec(simulation.RandomAmount(r, maxCommission).Int64(), 1)

		val := keeper.RandomValidator(r, k, ctx)
		address := val.GetOperator()
		msg := stake.MsgEditValidator{
			Description:    description,
			ValidatorAddr:  address,
			CommissionRate: &newCommissionRate,
		}

		if msg.ValidateBasic() != nil {
			return "", nil, fmt.Errorf("expected msg to pass ValidateBasic: %s", msg.GetSignBytes())
		}

		ctx, write := ctx.CacheContext()
		result := handler(ctx, msg)
		if result.IsOK() {
			write()
		}
		event(fmt.Sprintf("stake/MsgEditValidator/%v", result.IsOK()))
		action = fmt.Sprintf("TestMsgEditValidator: ok %v, msg %s", result.IsOK(), msg.GetSignBytes())
		return action, nil, nil
	}
}

// SimulateMsgDelegate
func SimulateMsgDelegate(m auth.AccountKeeper, k stake.Keeper) simulation.Operation {
	handler := stake.NewHandler(k)
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simulation.Account, event func(string)) (
		action string, fOp []simulation.FutureOperation, err error) {

		denom := k.BondDenom()
		val := keeper.RandomValidator(r, k, ctx)
		validatorAddress := val.GetOperator()
		delegatorAcc := simulation.RandomAcc(r, accs)
		delegatorAddress := delegatorAcc.Address
		amount := m.GetAccount(ctx, delegatorAddress).GetCoins().AmountOf(denom)

		if amount.Equal(sdk.ZeroInt()) {
			return "no-operation", nil, nil
		}
		msg := stake.MsgDelegate{
			DelegatorAddr: delegatorAddress,
			ValidatorAddr: validatorAddress,
			Delegation:    sdk.NewCoin(denom, amount),
		}
		if msg.ValidateBasic() != nil {
			return "", nil, fmt.Errorf("expected msg to pass ValidateBasic: %s", msg.GetSignBytes())
		}
		ctx, write := ctx.CacheContext()
		result := handler(ctx, msg)
		if result.IsOK() {
			write()
		}
		event(fmt.Sprintf("stake/MsgDelegate/%v", result.IsOK()))
		action = fmt.Sprintf("TestMsgDelegate: ok %v, msg %s", result.IsOK(), msg.GetSignBytes())
		return action, nil, nil
	}
}

// SimulateMsgBeginUnbonding
func SimulateMsgBeginUnbonding(m auth.AccountKeeper, k stake.Keeper) simulation.Operation {
	handler := stake.NewHandler(k)
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simulation.Account, event func(string)) (
		action string, fOp []simulation.FutureOperation, err error) {

		delegatorAcc := simulation.RandomAcc(r, accs)
		delegatorAddress := delegatorAcc.Address
		delegations := k.GetAllDelegatorDelegations(ctx, delegatorAddress)
		if len(delegations) == 0 {
			return "no-operation", nil, nil
		}
		delegation := delegations[r.Intn(len(delegations))]

		numShares := simulation.RandomDecAmount(r, delegation.Shares)
		if numShares.Equal(sdk.ZeroDec()) {
			return "no-operation", nil, nil
		}
		msg := stake.MsgBeginUnbonding{
			DelegatorAddr: delegatorAddress,
			ValidatorAddr: delegation.ValidatorAddr,
			SharesAmount:  numShares,
		}
		if msg.ValidateBasic() != nil {
			return "", nil, fmt.Errorf("expected msg to pass ValidateBasic: %s, got error %v",
				msg.GetSignBytes(), msg.ValidateBasic())
		}
		ctx, write := ctx.CacheContext()
		result := handler(ctx, msg)
		if result.IsOK() {
			write()
		}
		event(fmt.Sprintf("stake/MsgBeginUnbonding/%v", result.IsOK()))
		action = fmt.Sprintf("TestMsgBeginUnbonding: ok %v, msg %s", result.IsOK(), msg.GetSignBytes())
		return action, nil, nil
	}
}

// SimulateMsgBeginRedelegate
func SimulateMsgBeginRedelegate(m auth.AccountKeeper, k stake.Keeper) simulation.Operation {
	handler := stake.NewHandler(k)
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simulation.Account, event func(string)) (
		action string, fOp []simulation.FutureOperation, err error) {

		denom := k.BondDenom()
		srcVal := keeper.RandomValidator(r, k, ctx)
		srcValidatorAddress := srcVal.GetOperator()
		destVal := keeper.RandomValidator(r, k, ctx)
		destValidatorAddress := destVal.GetOperator()
		delegatorAcc := simulation.RandomAcc(r, accs)
		delegatorAddress := delegatorAcc.Address
		// TODO
		amount := m.GetAccount(ctx, delegatorAddress).GetCoins().AmountOf(denom)
		if amount.GT(sdk.ZeroInt()) {
			amount = simulation.RandomAmount(r, amount)
		}
		if amount.Equal(sdk.ZeroInt()) {
			return "no-operation", nil, nil
		}
		msg := stake.MsgBeginRedelegate{
			DelegatorAddr:    delegatorAddress,
			ValidatorSrcAddr: srcValidatorAddress,
			ValidatorDstAddr: destValidatorAddress,
			SharesAmount:     sdk.NewDecFromInt(amount),
		}
		if msg.ValidateBasic() != nil {
			return "", nil, fmt.Errorf("expected msg to pass ValidateBasic: %s", msg.GetSignBytes())
		}
		ctx, write := ctx.CacheContext()
		result := handler(ctx, msg)
		if result.IsOK() {
			write()
		}
		event(fmt.Sprintf("stake/MsgBeginRedelegate/%v", result.IsOK()))
		action = fmt.Sprintf("TestMsgBeginRedelegate: %s", msg.GetSignBytes())
		return action, nil, nil
	}
}

// Setup
// nolint: errcheck
func Setup(mapp *mock.App, k stake.Keeper) simulation.RandSetup {
	return func(r *rand.Rand, accs []simulation.Account) {
		ctx := mapp.NewContext(false, abci.Header{})
		gen := stake.DefaultGenesisState()
		stake.InitGenesis(ctx, k, gen)
		denom := k.BondDenom()
		loose := sdk.ZeroInt()
		mapp.AccountKeeper.IterateAccounts(ctx, func(acc auth.Account) bool {
			balance := simulation.RandomAmount(r, sdk.NewInt(1000000))
			acc.SetCoins(acc.GetCoins().Add(sdk.Coins{sdk.NewCoin(denom, balance)}))
			mapp.AccountKeeper.SetAccount(ctx, acc)
			loose = loose.Add(balance)
			return false
		})
		pool := k.GetPool(ctx)
		pool.BankKeeper.IncreaseLoosenToken(ctx, sdk.Coins{sdk.NewCoin(types.StakeDenom, loose)})
	}
}
