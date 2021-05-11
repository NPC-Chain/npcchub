package lcd

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/app/v1/stake"
	"github.com/NPC-Chain/npcchub/client/context"
	stakeClient "github.com/NPC-Chain/npcchub/client/stake"
	"github.com/NPC-Chain/npcchub/client/utils"
	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
)

func registerTxRoutes(cliCtx context.CLIContext, r *mux.Router, cdc *codec.Codec) {
	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/delegations",
		delegationsRequestHandlerFn(cdc, cliCtx),
	).Methods("POST")

	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/redelegations",
		beginRedelegatesRequestHandlerFn(cdc, cliCtx),
	).Methods("POST")

	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/unbonding-delegations",
		beginUnbondingRequestHandlerFn(cdc, cliCtx),
	).Methods("POST")
}

type (
	msgDelegateInput struct {
		ValidatorAddr string `json:"validator_addr"` // in bech32
		Delegation    string `json:"delegation"`
	}

	msgRedelegateInput struct {
		ValidatorSrcAddr string `json:"validator_src_addr"` // in bech32
		ValidatorDstAddr string `json:"validator_dst_addr"` // in bech32
		SharesAmount     string `json:"shares_amount"`
		SharesPercent    string `json:"shares_percent"`
	}

	msgUnbondInput struct {
		ValidatorAddr string `json:"validator_addr"` // in bech32
		SharesAmount  string `json:"shares_amount"`
		SharesPercent string `json:"shares_percent"`
	}

	// the request body for edit delegations
	DelegationsReq struct {
		BaseReq    utils.BaseTx     `json:"base_tx"`
		Delegation msgDelegateInput `json:"delegate"`
	}

	BeginUnbondingReq struct {
		BaseReq     utils.BaseTx   `json:"base_tx"`
		BeginUnbond msgUnbondInput `json:"unbond"`
	}

	BeginRedelegatesReq struct {
		BaseReq         utils.BaseTx       `json:"base_tx"`
		BeginRedelegate msgRedelegateInput `json:"redelegate"`
	}
)

// TODO: Split this up into several smaller functions, and remove the above nolint
// TODO: use sdk.ValAddress instead of sdk.AccAddress for validators in messages
// TODO: Seriously consider how to refactor...do we need to make it multiple txs?
// If not, we can just use CompleteAndBroadcastTxREST.
func delegationsRequestHandlerFn(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bech32delegator := vars["delegatorAddr"]

		var req DelegationsReq

		err := utils.ReadPostBody(w, r, cdc, &req)
		if err != nil {
			return
		}

		baseReq := req.BaseReq.Sanitize()
		if !baseReq.ValidateBasic(w) {
			return
		}

		// build messages
		delAddr, err := sdk.AccAddressFromBech32(bech32delegator)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		valAddr, err := sdk.ValAddressFromBech32(req.Delegation.ValidatorAddr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		delegationToken, err := cliCtx.ParseCoin(req.Delegation.Delegation)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		msg := stake.MsgDelegate{
			DelegatorAddr: delAddr,
			ValidatorAddr: valAddr,
			Delegation:    delegationToken}

		txCtx := utils.BuildReqTxCtx(cliCtx, baseReq, w)

		utils.WriteGenerateStdTxResponse(w, txCtx, []sdk.Msg{msg})
	}
}

// TODO: Split this up into several smaller functions, and remove the above nolint
// TODO: use sdk.ValAddress instead of sdk.AccAddress for validators in messages
// TODO: Seriously consider how to refactor...do we need to make it multiple txs?
// If not, we can just use CompleteAndBroadcastTxREST.
func beginRedelegatesRequestHandlerFn(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bech32delegator := vars["delegatorAddr"]

		var req BeginRedelegatesReq

		err := utils.ReadPostBody(w, r, cdc, &req)
		if err != nil {
			return
		}

		baseReq := req.BaseReq.Sanitize()
		if !baseReq.ValidateBasic(w) {
			return
		}

		delAddr, err := sdk.AccAddressFromBech32(bech32delegator)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		valSrcAddr, err := sdk.ValAddressFromBech32(req.BeginRedelegate.ValidatorSrcAddr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		valDstAddr, err := sdk.ValAddressFromBech32(req.BeginRedelegate.ValidatorDstAddr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		sharesAmount, err := stakeClient.GetShares(
			storeName, cliCtx, cdc, req.BeginRedelegate.SharesAmount, req.BeginRedelegate.SharesPercent,
			delAddr, valSrcAddr,
		)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := stake.MsgBeginRedelegate{
			DelegatorAddr:    delAddr,
			ValidatorSrcAddr: valSrcAddr,
			ValidatorDstAddr: valDstAddr,
			SharesAmount:     sharesAmount,
		}

		txCtx := utils.BuildReqTxCtx(cliCtx, baseReq, w)

		utils.WriteGenerateStdTxResponse(w, txCtx, []sdk.Msg{msg})
	}
}

// TODO: Split this up into several smaller functions, and remove the above nolint
// TODO: use sdk.ValAddress instead of sdk.AccAddress for validators in messages
// TODO: Seriously consider how to refactor...do we need to make it multiple txs?
// If not, we can just use CompleteAndBroadcastTxREST.
func beginUnbondingRequestHandlerFn(cdc *codec.Codec, cliCtx context.CLIContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		bech32delegator := vars["delegatorAddr"]

		var req BeginUnbondingReq

		err := utils.ReadPostBody(w, r, cdc, &req)
		if err != nil {
			return
		}

		baseReq := req.BaseReq.Sanitize()
		if !baseReq.ValidateBasic(w) {
			return
		}

		delAddr, err := sdk.AccAddressFromBech32(bech32delegator)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		valAddr, err := sdk.ValAddressFromBech32(req.BeginUnbond.ValidatorAddr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		sharesAmount, err := stakeClient.GetShares(
			storeName, cliCtx, cdc, req.BeginUnbond.SharesAmount, req.BeginUnbond.SharesPercent,
			delAddr, valAddr,
		)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		msg := stake.MsgBeginUnbonding{
			DelegatorAddr: delAddr,
			ValidatorAddr: valAddr,
			SharesAmount:  sharesAmount,
		}

		txCtx := utils.BuildReqTxCtx(cliCtx, baseReq, w)

		utils.WriteGenerateStdTxResponse(w, txCtx, []sdk.Msg{msg})
	}
}
