package lcd

import (
	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/codec"
)

// RegisterRoutes registers staking-related REST handlers to a router
func RegisterRoutes(cliCtx context.CLIContext, r *mux.Router, cdc *codec.Codec) {
	r.HandleFunc("/slashing/validators/{validatorPubKey}/signing-info",
		signingInfoHandlerFn(cliCtx, "slashing", cdc)).Methods("GET")
	r.HandleFunc("/slashing/validators/{validatorAddr}/unjail",
		unrevokeRequestHandlerFn(cdc, cliCtx)).Methods("POST")
}
