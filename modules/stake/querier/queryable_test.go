package querier

import (
	"testing"

	"github.com/NPC-Chain/npcchub/codec"
	keep "github.com/NPC-Chain/npcchub/modules/stake/keeper"
	"github.com/NPC-Chain/npcchub/modules/stake/types"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	addrAcc1, addrAcc2 = keep.Addrs[0], keep.Addrs[1]
	addrVal1, addrVal2 = sdk.ValAddress(keep.Addrs[0]), sdk.ValAddress(keep.Addrs[1])
	pk1, pk2           = keep.PKs[0], keep.PKs[1]
)

func TestNewQuerier(t *testing.T) {
	cdc := codec.New()
	ctx, _, keeper := keep.CreateTestInput(t, false, sdk.NewIntWithDecimal(1000, 18))
	pool := keeper.GetPool(ctx)
	// Create Validators
	amts := []sdk.Int{sdk.NewInt(9), sdk.NewInt(8)}
	var validators [2]types.Validator
	for i, amt := range amts {
		validators[i] = types.NewValidator(sdk.ValAddress(keep.Addrs[i]), keep.PKs[i], types.Description{})
		validators[i], pool, _ = validators[i].AddTokensFromDel(ctx, pool, amt)
		keeper.SetValidator(ctx, validators[i])
		keeper.SetValidatorByPowerIndex(ctx, validators[i], pool)
	}
	keeper.SetPool(ctx, pool)

	query := abci.RequestQuery{
		Path: "",
		Data: []byte{},
	}

	querier := NewQuerier(keeper, cdc)

	bz, err := querier(ctx, []string{"other"}, query)
	require.NotNil(t, err)
	require.Nil(t, bz)

	paginationParams := sdk.NewPaginationParams(0, 1)
	bz, errRes := cdc.MarshalJSON(paginationParams)
	require.Nil(t, errRes)

	query.Path = "/custom/stake/validators"
	query.Data = bz
	_, err = querier(ctx, []string{"validators"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"pool"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"parameters"}, query)
	require.Nil(t, err)

	queryValParams := NewQueryValidatorParams(addrVal1)
	bz, errRes = cdc.MarshalJSON(queryValParams)
	require.Nil(t, errRes)

	query.Path = "/custom/stake/validator"
	query.Data = bz

	_, err = querier(ctx, []string{"validator"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"validatorUnbondingDelegations"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"validatorRedelegations"}, query)
	require.Nil(t, err)

	queryDelParams := NewQueryDelegatorParams(addrAcc2)
	bz, errRes = cdc.MarshalJSON(queryDelParams)
	require.Nil(t, errRes)

	query.Path = "/custom/stake/validator"
	query.Data = bz

	_, err = querier(ctx, []string{"delegatorDelegations"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"delegatorUnbondingDelegations"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"delegatorRedelegations"}, query)
	require.Nil(t, err)

	_, err = querier(ctx, []string{"delegatorValidators"}, query)
	require.Nil(t, err)
}

func TestQueryParametersPool(t *testing.T) {
	cdc := codec.New()
	ctx, _, keeper := keep.CreateTestInput(t, false, sdk.NewIntWithDecimal(1000, 18))

	res, err := queryParameters(ctx, cdc, keeper)
	require.Nil(t, err)

	var params types.Params
	errRes := cdc.UnmarshalJSON(res, &params)
	require.Nil(t, errRes)
	require.Equal(t, keeper.GetParams(ctx), params)

	res, err = queryPool(ctx, cdc, keeper)
	require.Nil(t, err)

	var poolStatus types.PoolStatus
	errRes = cdc.UnmarshalJSON(res, &poolStatus)
	require.Nil(t, errRes)
	require.Equal(t, keeper.GetPool(ctx).BondedPool.BondedTokens, poolStatus.BondedTokens)
	require.Equal(t, keeper.GetPool(ctx).GetLoosenTokenAmount(ctx), poolStatus.LooseTokens)
}

func TestQueryValidators(t *testing.T) {
	cdc := codec.New()
	ctx, _, keeper := keep.CreateTestInput(t, false, sdk.NewIntWithDecimal(10000, 18))
	pool := keeper.GetPool(ctx)
	params := keeper.GetParams(ctx)

	// Create Validators
	amts := []sdk.Int{sdk.NewInt(9), sdk.NewInt(8)}
	var validators [2]types.Validator
	for i, amt := range amts {
		validators[i] = types.NewValidator(sdk.ValAddress(keep.Addrs[i]), keep.PKs[i], types.Description{})
		validators[i], pool, _ = validators[i].AddTokensFromDel(ctx, pool, amt)
	}
	keeper.SetPool(ctx, pool)
	keeper.SetValidator(ctx, validators[0])
	keeper.SetValidator(ctx, validators[1])

	// Query Validators
	queriedValidators := keeper.GetValidators(ctx, 0, params.MaxValidators)

	paginationParams := sdk.NewPaginationParams(0, params.MaxValidators)
	bz, errRes := cdc.MarshalJSON(paginationParams)
	query := abci.RequestQuery{
		Path: "/custom/stake/validators",
		Data: bz,
	}
	res, err := queryValidators(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var validatorsResp []types.Validator
	errRes = cdc.UnmarshalJSON(res, &validatorsResp)
	require.Nil(t, errRes)

	require.Equal(t, len(queriedValidators), len(validatorsResp))
	require.ElementsMatch(t, queriedValidators, validatorsResp)

	// Query each validator
	queryParams := NewQueryValidatorParams(addrVal1)
	bz, errRes = cdc.MarshalJSON(queryParams)
	require.Nil(t, errRes)

	query = abci.RequestQuery{
		Path: "/custom/stake/validator",
		Data: bz,
	}
	res, err = queryValidator(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var validator types.Validator
	errRes = cdc.UnmarshalJSON(res, &validator)
	require.Nil(t, errRes)

	require.Equal(t, queriedValidators[0], validator)
}

func TestQueryDelegation(t *testing.T) {
	cdc := codec.New()
	ctx, _, keeper := keep.CreateTestInput(t, false, sdk.NewIntWithDecimal(10000, 18))
	params := keeper.GetParams(ctx)

	// Create Validators and Delegation
	val1 := types.NewValidator(addrVal1, pk1, types.Description{})
	keeper.SetValidator(ctx, val1)
	pool := keeper.GetPool(ctx)
	keeper.SetValidatorByPowerIndex(ctx, val1, pool)

	val2 := types.NewValidator(addrVal2, pk2, types.Description{})
	keeper.SetValidator(ctx, val2)
	pool = keeper.GetPool(ctx)
	keeper.SetValidatorByPowerIndex(ctx, val2, pool)

	keeper.Delegate(ctx, addrAcc2, sdk.NewCoin(types.StakeDenom, sdk.NewIntWithDecimal(20, 18)), val1, true)

	// apply TM updates
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)

	// Query Delegator bonded validators
	queryParams := NewQueryDelegatorParams(addrAcc2)
	bz, errRes := cdc.MarshalJSON(queryParams)
	require.Nil(t, errRes)

	query := abci.RequestQuery{
		Path: "/custom/stake/delegatorValidators",
		Data: bz,
	}

	delValidators := keeper.GetDelegatorValidators(ctx, addrAcc2, params.MaxValidators)

	res, err := queryDelegatorValidators(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var validatorsResp []types.Validator
	errRes = cdc.UnmarshalJSON(res, &validatorsResp)
	require.Nil(t, errRes)

	require.Equal(t, len(delValidators), len(validatorsResp))
	require.ElementsMatch(t, delValidators, validatorsResp)

	// error unknown request
	query.Data = bz[:len(bz)-1]

	_, err = queryDelegatorValidators(ctx, cdc, query, keeper)
	require.NotNil(t, err)

	// Query bonded validator
	queryBondParams := NewQueryBondsParams(addrAcc2, addrVal1)
	bz, errRes = cdc.MarshalJSON(queryBondParams)
	require.Nil(t, errRes)

	query = abci.RequestQuery{
		Path: "/custom/stake/delegatorValidator",
		Data: bz,
	}

	res, err = queryDelegatorValidator(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var validator types.Validator
	errRes = cdc.UnmarshalJSON(res, &validator)
	require.Nil(t, errRes)

	require.Equal(t, delValidators[0], validator)

	// error unknown request
	query.Data = bz[:len(bz)-1]

	_, err = queryDelegatorValidator(ctx, cdc, query, keeper)
	require.NotNil(t, err)

	// Query delegation

	query = abci.RequestQuery{
		Path: "/custom/stake/delegation",
		Data: bz,
	}

	delegation, found := keeper.GetDelegation(ctx, addrAcc2, addrVal1)
	require.True(t, found)

	res, err = queryDelegation(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var delegationRes types.Delegation
	errRes = cdc.UnmarshalJSON(res, &delegationRes)
	require.Nil(t, errRes)

	require.Equal(t, delegation, delegationRes)

	// Query Delegator Delegations

	query = abci.RequestQuery{
		Path: "/custom/stake/delegatorDelegations",
		Data: bz,
	}

	res, err = queryDelegatorDelegations(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var delegatorDelegations []types.Delegation
	errRes = cdc.UnmarshalJSON(res, &delegatorDelegations)
	require.Nil(t, errRes)
	require.Len(t, delegatorDelegations, 1)
	require.Equal(t, delegation, delegatorDelegations[0])

	// error unknown request
	query.Data = bz[:len(bz)-1]

	_, err = queryDelegation(ctx, cdc, query, keeper)
	require.NotNil(t, err)

	// Query validator delegations
	bz, errRes = cdc.MarshalJSON(NewQueryValidatorParams(addrVal1))
	require.Nil(t, errRes)
	query = abci.RequestQuery{
		Path: "custom/stake/validatorDelegations",
		Data: bz,
	}
	res, err = queryValidatorDelegations(ctx, cdc, query, keeper)
	require.Nil(t, err)
	var delegationsRes []types.Delegation
	errRes = cdc.UnmarshalJSON(res, &delegationsRes)
	require.Nil(t, errRes)
	require.Equal(t, delegationsRes[0], delegation)

	// Query unbonging delegation
	keeper.BeginUnbonding(ctx, addrAcc2, val1.OperatorAddr, sdk.NewDec(10))

	queryBondParams = NewQueryBondsParams(addrAcc2, addrVal1)
	bz, errRes = cdc.MarshalJSON(queryBondParams)
	require.Nil(t, errRes)

	query = abci.RequestQuery{
		Path: "/custom/stake/unbondingDelegation",
		Data: bz,
	}

	unbond, found := keeper.GetUnbondingDelegation(ctx, addrAcc2, addrVal1)
	require.True(t, found)

	res, err = queryUnbondingDelegation(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var unbondRes types.UnbondingDelegation
	errRes = cdc.UnmarshalJSON(res, &unbondRes)
	require.Nil(t, errRes)

	require.Equal(t, unbond, unbondRes)

	// error unknown request
	query.Data = bz[:len(bz)-1]

	_, err = queryUnbondingDelegation(ctx, cdc, query, keeper)
	require.NotNil(t, err)

	// Query Delegator Delegations

	query = abci.RequestQuery{
		Path: "/custom/stake/delegatorUnbondingDelegations",
		Data: bz,
	}

	res, err = queryDelegatorUnbondingDelegations(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var delegatorUbds []types.UnbondingDelegation
	errRes = cdc.UnmarshalJSON(res, &delegatorUbds)
	require.Nil(t, errRes)
	require.Equal(t, unbond, delegatorUbds[0])

	// error unknown request
	query.Data = bz[:len(bz)-1]

	_, err = queryDelegatorUnbondingDelegations(ctx, cdc, query, keeper)
	require.NotNil(t, err)
}

func TestQueryRedelegations(t *testing.T) {
	cdc := codec.New()
	ctx, _, keeper := keep.CreateTestInput(t, false, sdk.NewIntWithDecimal(10000, 18))

	// Create Validators and Delegation
	val1 := types.NewValidator(addrVal1, pk1, types.Description{})
	val2 := types.NewValidator(addrVal2, pk2, types.Description{})
	keeper.SetValidator(ctx, val1)
	keeper.SetValidator(ctx, val2)

	keeper.Delegate(ctx, addrAcc2, sdk.NewCoin(types.StakeDenom, sdk.NewIntWithDecimal(100, 18)), val1, true)
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)

	keeper.BeginRedelegation(ctx, addrAcc2, val1.GetOperator(), val2.GetOperator(), sdk.NewDecFromInt(sdk.NewIntWithDecimal(20, 18)))
	keeper.ApplyAndReturnValidatorSetUpdates(ctx)

	redelegation, found := keeper.GetRedelegation(ctx, addrAcc2, val1.OperatorAddr, val2.OperatorAddr)
	require.True(t, found)

	// delegator redelegations
	queryDelegatorParams := NewQueryDelegatorParams(addrAcc2)
	bz, errRes := cdc.MarshalJSON(queryDelegatorParams)
	require.Nil(t, errRes)

	query := abci.RequestQuery{
		Path: "/custom/stake/delegatorRedelegations",
		Data: bz,
	}

	res, err := queryDelegatorRedelegations(ctx, cdc, query, keeper)
	require.Nil(t, err)

	var redsRes []types.Redelegation
	errRes = cdc.UnmarshalJSON(res, &redsRes)
	require.Nil(t, errRes)

	require.Equal(t, redelegation, redsRes[0])

	// validator redelegations
	queryValidatorParams := NewQueryValidatorParams(val1.GetOperator())
	bz, errRes = cdc.MarshalJSON(queryValidatorParams)
	require.Nil(t, errRes)

	query = abci.RequestQuery{
		Path: "/custom/stake/validatorRedelegations",
		Data: bz,
	}

	res, err = queryValidatorRedelegations(ctx, cdc, query, keeper)
	require.Nil(t, err)

	errRes = cdc.UnmarshalJSON(res, &redsRes)
	require.Nil(t, errRes)

	require.Equal(t, redelegation, redsRes[0])
}
