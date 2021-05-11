package lcd

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/app/protocol"
	"github.com/NPC-Chain/npcchub/app/v2/htlc"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/client/utils"
	"github.com/NPC-Chain/npcchub/codec"
)

func registerQueryRoutes(cliCtx context.CLIContext, r *mux.Router, cdc *codec.Codec) {
	// Get the HTLC by the hash lock
	r.HandleFunc(
		"/htlc/htlcs/{hash-lock}",
		queryHTLCHandlerFn(cliCtx, cdc),
	).Methods("GET")
}

func queryHTLCHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		hashLockStr := vars["hash-lock"]
		hashLock, err := hex.DecodeString(hashLockStr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		params := htlc.QueryHTLCParams{
			HashLock: hashLock,
		}

		bz, err := cliCtx.Codec.MarshalJSON(params)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		res, err := cliCtx.QueryWithData(fmt.Sprintf("custom/%s/%s", protocol.HtlcRoute, htlc.QueryHTLC), bz)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		utils.PostProcessResponse(w, cliCtx.Codec, res, cliCtx.Indent)
	}
}
