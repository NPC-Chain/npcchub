package lcd

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/app/v1/slashing"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/client/utils"
	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
)

// Unrevoke TX body
type UnjailBody struct {
	BaseTx utils.BaseTx `json:"base_tx"`
}

func unrevokeRequestHandlerFn(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		validatorAddr, err := sdk.ValAddressFromBech32(vars["validatorAddr"])
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		var m UnjailBody
		err = utils.ReadPostBody(w, r, cdc, &m)
		if err != nil {
			return
		}

		baseReq := m.BaseTx.Sanitize()
		if !baseReq.ValidateBasic(w) {
			return
		}

		msg := slashing.NewMsgUnjail(validatorAddr)

		txCtx := utils.BuildReqTxCtx(cliCtx, baseReq, w)

		utils.WriteGenerateStdTxResponse(w, txCtx, []sdk.Msg{msg})
	}
}
