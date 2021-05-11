package lcd

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/app/v1/slashing"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/client/utils"
	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
	"net/http"
)

// http request handler to query signing info
func signingInfoHandlerFn(cliCtx context.CLIContext, storeName string, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		pk, err := sdk.GetConsPubKeyBech32(vars["validatorPubKey"])
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		key := slashing.GetValidatorSigningInfoKey(sdk.ConsAddress(pk.Address()))

		res, err := cliCtx.QueryStore(key, storeName)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("couldn't query signing info. Error: %s", err.Error()))
			return
		}
		if len(res) == 0 {
			utils.WriteErrorResponse(w, http.StatusNoContent, "")
			return
		}

		var signingInfo slashing.ValidatorSigningInfo

		err = cdc.UnmarshalBinaryLengthPrefixed(res, &signingInfo)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("couldn't decode signing info. Error: %s", err.Error()))
			return
		}

		utils.PostProcessResponse(w, cliCtx.Codec, signingInfo, cliCtx.Indent)
	}
}
