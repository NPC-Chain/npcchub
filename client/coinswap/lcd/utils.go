package lcd

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/app/protocol"
	"github.com/NPC-Chain/npcchub/app/v2/coinswap"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/client/utils"
	"github.com/NPC-Chain/npcchub/codec"
)

func queryLiquidity(cliCtx context.CLIContext, cdc *codec.Codec, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]
		params := coinswap.QueryLiquidityParams{
			Id: id,
		}

		bz, err := cliCtx.Codec.MarshalJSON(params)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		res, err := cliCtx.QueryWithData(
			fmt.Sprintf("custom/%s/%s", protocol.SwapRoute, coinswap.QueryLiquidity), bz)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		utils.PostProcessResponse(w, cliCtx.Codec, res, cliCtx.Indent)
	}
}
