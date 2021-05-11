package lcd

import (
	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/client/utils"
	"github.com/NPC-Chain/npcchub/codec"
)

// RegisterRoutes - Central function to define routes that get registered by the main application
func RegisterRoutes(cliCtx context.CLIContext, r *mux.Router, cdc *codec.Codec) {
	r.HandleFunc("/bank/accounts/{address}",
		QueryAccountRequestHandlerFn(cdc, utils.GetAccountDecoder(cdc), cliCtx)).Methods("GET")
	r.HandleFunc("/bank/coins/{type}",
		QueryCoinTypeRequestHandlerFn(cdc, cliCtx)).Methods("GET")
	r.HandleFunc("/bank/token-stats",
		QueryTokenStatsRequestHandlerFn(cdc, utils.GetAccountDecoder(cdc), cliCtx)).Methods("GET")
	r.HandleFunc("/bank/token-stats/{id}",
		QueryTokenStatsRequestHandlerFn(cdc, utils.GetAccountDecoder(cdc), cliCtx)).Methods("GET")
	r.HandleFunc("/bank/accounts/{address}/send", SendRequestHandlerFn(cdc, cliCtx)).Methods("POST")
	r.HandleFunc("/bank/accounts/{address}/burn", BurnRequestHandlerFn(cdc, cliCtx)).Methods("POST")
	r.HandleFunc("/bank/accounts/{address}/set-memo-regexp", SetMemoRegexpRequestHandlerFn(cdc, cliCtx)).Methods("POST")
}
