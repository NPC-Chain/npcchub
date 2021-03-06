package stake

import (
	"testing"

	"github.com/NPC-Chain/npcchub/mock"
	"github.com/NPC-Chain/npcchub/modules/auth"
	"github.com/NPC-Chain/npcchub/modules/bank"
	"github.com/NPC-Chain/npcchub/modules/params"
	"github.com/NPC-Chain/npcchub/modules/stake/types"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

// getMockApp returns an initialized mock application for this module.
func getMockApp(t *testing.T) (*mock.App, Keeper) {
	mApp := mock.NewApp()

	RegisterCodec(mApp.Cdc)

	bankKeeper := bank.NewBaseKeeper(mApp.AccountKeeper)
	pk := params.NewKeeper(mApp.Cdc, mApp.KeyParams, mApp.TkeyParams)

	keeper := NewKeeper(mApp.Cdc, mApp.KeyStake, mApp.TkeyStake, bankKeeper, pk.Subspace(DefaultParamspace), DefaultCodespace, NopMetrics())

	mApp.Router().AddRoute("stake", []*sdk.KVStoreKey{mApp.KeyStake, mApp.KeyAccount, mApp.KeyParams}, NewHandler(keeper))
	mApp.SetEndBlocker(getEndBlocker(keeper))
	mApp.SetInitChainer(getInitChainer(mApp, keeper))

	require.NoError(t, mApp.CompleteSetup())

	return mApp, keeper
}

// getEndBlocker returns a stake endblocker.
func getEndBlocker(keeper Keeper) sdk.EndBlocker {
	return func(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
		validatorUpdates := EndBlocker(ctx, keeper)

		return abci.ResponseEndBlock{
			ValidatorUpdates: validatorUpdates,
		}
	}
}

// getInitChainer initializes the chainer of the mock app and sets the genesis
// state. It returns an empty ResponseInitChain.
func getInitChainer(mapp *mock.App, keeper Keeper) sdk.InitChainer {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		mapp.InitChainer(ctx, req)

		stakeGenesis := DefaultGenesisState()

		validators, err := InitGenesis(ctx, keeper, stakeGenesis)
		if err != nil {
			panic(err)
		}
		return abci.ResponseInitChain{
			Validators: validators,
		}
	}
}

//__________________________________________________________________________________________

func checkValidator(t *testing.T, mapp *mock.App, keeper Keeper,
	addr sdk.ValAddress, expFound bool) Validator {

	ctxCheck := mapp.BaseApp.NewContext(true, abci.Header{})
	validator, found := keeper.GetValidator(ctxCheck, addr)

	require.Equal(t, expFound, found)
	return validator
}

func checkDelegation(
	t *testing.T, mapp *mock.App, keeper Keeper, delegatorAddr sdk.AccAddress,
	validatorAddr sdk.ValAddress, expFound bool, expShares sdk.Dec,
) {

	ctxCheck := mapp.BaseApp.NewContext(true, abci.Header{})
	delegation, found := keeper.GetDelegation(ctxCheck, delegatorAddr, validatorAddr)
	if expFound {
		require.True(t, found)
		require.True(sdk.DecEq(t, expShares, delegation.Shares))

		return
	}

	require.False(t, found)
}

func TestStakeMsgs(t *testing.T) {
	mApp, keeper := getMockApp(t)

	genCoin := sdk.NewCoin(types.StakeDenom, sdk.NewIntWithDecimal(42, 18))
	bondCoin := sdk.NewCoin(types.StakeDenom, sdk.NewIntWithDecimal(10, 18))

	acc1 := &auth.BaseAccount{
		Address: addr1,
		Coins:   sdk.Coins{genCoin},
	}
	acc2 := &auth.BaseAccount{
		Address: addr2,
		Coins:   sdk.Coins{genCoin},
	}
	accs := []auth.Account{acc1, acc2}

	mock.SetGenesis(mApp, accs)
	mock.CheckBalance(t, mApp, addr1, sdk.Coins{genCoin})
	mock.CheckBalance(t, mApp, addr2, sdk.Coins{genCoin})

	// create validator
	description := NewDescription("foo_moniker", "", "", "")
	createValidatorMsg := NewMsgCreateValidator(
		sdk.ValAddress(addr1), priv1.PubKey(), bondCoin, description, commissionMsg,
	)

	mock.SignCheckDeliver(t, mApp.BaseApp, []sdk.Msg{createValidatorMsg}, []uint64{0}, []uint64{0}, true, true, priv1)
	mock.CheckBalance(t, mApp, addr1, sdk.Coins{genCoin.Sub(bondCoin)})
	mApp.BeginBlock(abci.RequestBeginBlock{})

	validator := checkValidator(t, mApp, keeper, sdk.ValAddress(addr1), true)
	require.Equal(t, sdk.ValAddress(addr1), validator.OperatorAddr)
	require.Equal(t, sdk.Bonded, validator.Status)
	require.True(sdk.DecEq(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)), validator.BondedTokens()))

	// addr1 create validator on behalf of addr2
	createValidatorMsgOnBehalfOf := NewMsgCreateValidatorOnBehalfOf(
		addr1, sdk.ValAddress(addr2), priv2.PubKey(), bondCoin, description, commissionMsg,
	)

	mock.SignCheckDeliver(t, mApp.BaseApp, []sdk.Msg{createValidatorMsgOnBehalfOf}, []uint64{0, 0}, []uint64{1, 0}, true, true, priv1, priv2)
	mock.CheckBalance(t, mApp, addr1, sdk.Coins{genCoin.Sub(bondCoin).Sub(bondCoin)})
	mApp.BeginBlock(abci.RequestBeginBlock{})

	validator = checkValidator(t, mApp, keeper, sdk.ValAddress(addr2), true)
	require.Equal(t, sdk.ValAddress(addr2), validator.OperatorAddr)
	require.Equal(t, sdk.Bonded, validator.Status)
	require.True(sdk.DecEq(t, sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)), validator.Tokens))

	// check the bond that should have been created as well
	checkDelegation(t, mApp, keeper, addr1, sdk.ValAddress(addr1), true, sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)))

	// edit the validator
	description = NewDescription("bar_moniker", "", "", "")
	editValidatorMsg := NewMsgEditValidator(sdk.ValAddress(addr1), description, nil)

	mock.SignCheckDeliver(t, mApp.BaseApp, []sdk.Msg{editValidatorMsg}, []uint64{0}, []uint64{2}, true, true, priv1)
	validator = checkValidator(t, mApp, keeper, sdk.ValAddress(addr1), true)
	require.Equal(t, description, validator.Description)

	// delegate
	mock.CheckBalance(t, mApp, addr2, sdk.Coins{genCoin})
	delegateMsg := NewMsgDelegate(addr2, sdk.ValAddress(addr1), bondCoin)

	mock.SignCheckDeliver(t, mApp.BaseApp, []sdk.Msg{delegateMsg}, []uint64{0}, []uint64{1}, true, true, priv2)
	mock.CheckBalance(t, mApp, addr2, sdk.Coins{genCoin.Sub(bondCoin)})
	checkDelegation(t, mApp, keeper, addr2, sdk.ValAddress(addr1), true, sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)))

	// begin unbonding
	beginUnbondingMsg := NewMsgBeginUnbonding(addr2, sdk.ValAddress(addr1), sdk.NewDecFromInt(sdk.NewIntWithDecimal(10, 18)))
	mock.SignCheckDeliver(t, mApp.BaseApp, []sdk.Msg{beginUnbondingMsg}, []uint64{0}, []uint64{2}, true, true, priv2)

	// delegation should exist anymore
	checkDelegation(t, mApp, keeper, addr2, sdk.ValAddress(addr1), false, sdk.Dec{})

	// balance should be the same because bonding not yet complete
	mock.CheckBalance(t, mApp, addr2, sdk.Coins{genCoin.Sub(bondCoin)})
}
