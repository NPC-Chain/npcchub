package params

import (
	"fmt"

	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// query endpoints supported by the params Querier
const (
	QueryModule = "module"
)

// creates a querier for params REST endpoints
func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, sdk.Error) {
		switch path[0] {
		case QueryModule:
			return queryModule(ctx, req, keeper)

		default:
			return nil, sdk.ErrUnknownRequest("unknown params query endpoint")
		}
	}
}

// defines the params for query: "custom/params/module"
type QueryModuleParams struct {
	Module string
}

func queryModule(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var params QueryModuleParams
	if err := keeper.cdc.UnmarshalJSON(req.Data, &params); err != nil {
		return nil, sdk.ParseParamsErr(err)
	}

	subspace, ok := keeper.GetSubspace(params.Module)
	if !ok {
		return nil, sdk.NewError(DefaultCodespace, CodeInvalidModule, fmt.Sprintf("The module %s is not existed or does not support params change", params.Module))
	}

	ps, ok := keeper.GetParamSet(params.Module)
	if !ok {
		return nil, sdk.NewError(DefaultCodespace, CodeInvalidModule, fmt.Sprintf("The module %s is does not support params change", params.Module))
	}

	subspace.GetParamSet(ctx, ps)
	bz, err := codec.MarshalJSONIndent(keeper.cdc, ps)
	if err != nil {
		return nil, sdk.MarshalResultErr(err)
	}

	return bz, nil
}
