package lcd

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/codec"
)

func registerQueryRoutes(cliCtx context.CLIContext, r *mux.Router, cdc *codec.Codec) {
	// Query liquidity
	r.HandleFunc(
		"/coinswap/liquidities/{id}",
		queryLiquidityHandlerFn(cliCtx, cdc),
	).Methods("GET")
}

// queryLiquidityHandlerFn performs liquidity information query
func queryLiquidityHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryLiquidity(cliCtx, cdc, "custom/coinswap/liquidities/{id}")
}
