package lcd

import (
	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/codec"
)

// RegisterRoutes - Central function to define routes that get registered by the main application
func RegisterRoutes(cliCtx context.CLIContext, r *mux.Router, cdc *codec.Codec) {
	r.HandleFunc("/distribution/{delegatorAddr}/withdraw-address", SetWithdrawAddressHandlerFn(cdc, cliCtx)).Methods("POST")
	r.HandleFunc("/distribution/{delegatorAddr}/rewards/withdraw", WithdrawRewardsHandlerFn(cdc, cliCtx)).Methods("POST")

	r.HandleFunc("/distribution/{delegatorAddr}/withdraw-address",
		QueryWithdrawAddressHandlerFn(cliCtx)).Methods("GET")
	r.HandleFunc("/distribution/{address}/rewards",
		QueryRewardsHandlerFn(cliCtx)).Methods("GET")
}
