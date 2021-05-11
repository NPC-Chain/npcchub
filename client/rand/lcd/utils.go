package lcd

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/app/protocol"
	"github.com/NPC-Chain/npcchub/app/v1/rand"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/client/rand/types"
	"github.com/NPC-Chain/npcchub/client/utils"
	"github.com/NPC-Chain/npcchub/codec"
)

func queryRand(cliCtx context.CLIContext, cdc *codec.Codec, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		reqID := vars["request-id"]
		if err := rand.CheckReqID(reqID); err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		params := rand.QueryRandParams{
			ReqID: reqID,
		}

		bz, err := cliCtx.Codec.MarshalJSON(params)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		res, err := cliCtx.QueryWithData(
			fmt.Sprintf("custom/%s/%s", protocol.RandRoute, rand.QueryRand), bz)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		var rawRand rand.Rand
		err = cdc.UnmarshalJSON(res, &rawRand)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		readableRand := types.ReadableRand{
			RequestTxHash: hex.EncodeToString(rawRand.RequestTxHash),
			Height:        rawRand.Height,
			Value:         rawRand.Value.Rat.FloatString(rand.RandPrec),
		}

		utils.PostProcessResponse(w, cliCtx.Codec, readableRand, cliCtx.Indent)
	}
}

func queryQueue(cliCtx context.CLIContext, cdc *codec.Codec, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		heightStr := r.FormValue("height")

		var (
			height int64
			err    error
		)

		if len(heightStr) != 0 {
			height, err = strconv.ParseInt(heightStr, 10, 64)
			if err != nil {
				utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
				return
			}

			if height < 0 {
				utils.WriteErrorResponse(w, http.StatusBadRequest, "the height must not be less than 0")
				return
			}
		}

		params := rand.QueryRandRequestQueueParams{
			Height: height,
		}

		bz, err := cliCtx.Codec.MarshalJSON(params)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		res, err := cliCtx.QueryWithData(
			fmt.Sprintf("custom/%s/%s", protocol.RandRoute, rand.QueryRandRequestQueue), bz)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		utils.PostProcessResponse(w, cliCtx.Codec, res, cliCtx.Indent)
	}
}
