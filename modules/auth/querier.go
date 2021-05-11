package auth

import (
	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// query endpoints supported by the auth Querier
const (
	QueryAccount    = "account"
	QueryTokenStats = "tokenStats"
)

// creates a querier for auth REST endpoints
func NewQuerier(keeper AccountKeeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, sdk.Error) {
		switch path[0] {
		case QueryAccount:
			return queryAccount(ctx, req, keeper)
		case QueryTokenStats:
			return queryTokenStats(ctx, keeper)

		default:
			return nil, sdk.ErrUnknownRequest("unknown auth query endpoint")
		}
	}
}

// defines the params for query: "custom/acc/account"
type QueryAccountParams struct {
	Address sdk.AccAddress
}

func NewQueryAccountParams(addr sdk.AccAddress) QueryAccountParams {
	return QueryAccountParams{
		Address: addr,
	}
}

func queryAccount(ctx sdk.Context, req abci.RequestQuery, keeper AccountKeeper) ([]byte, sdk.Error) {
	var params QueryAccountParams
	if err := keeper.cdc.UnmarshalJSON(req.Data, &params); err != nil {
		return nil, sdk.ParseParamsErr(err)
	}

	account := keeper.GetAccount(ctx, params.Address)
	if account == nil {
		return nil, sdk.ErrUnknownAddress(params.Address.String())
	}

	bz, err := codec.MarshalJSONIndent(keeper.cdc, account)
	if err != nil {
		return nil, sdk.MarshalResultErr(err)
	}

	return bz, nil
}

func queryTokenStats(ctx sdk.Context, keeper AccountKeeper) ([]byte, sdk.Error) {
	tokenStats := TokenStats{
		LooseTokens:  keeper.GetTotalLoosenToken(ctx),
		BurnedTokens: keeper.GetBurnedToken(ctx),
	}
	bz, err := codec.MarshalJSONIndent(keeper.cdc, tokenStats)
	if err != nil {
		return nil, sdk.MarshalResultErr(err)
	}

	return bz, nil
}

type TokenStats struct {
	LooseTokens  sdk.Coins `json:"loose_tokens"`
	BurnedTokens sdk.Coins `json:"burned_tokens"`
}
