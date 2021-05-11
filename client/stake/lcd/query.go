package lcd

import (
	"github.com/gorilla/mux"
	"github.com/NPC-Chain/npcchub/app/v1/stake/tags"
	"github.com/NPC-Chain/npcchub/app/v1/stake/types"
	"github.com/NPC-Chain/npcchub/client/context"
	stakeClient "github.com/NPC-Chain/npcchub/client/stake"
	"github.com/NPC-Chain/npcchub/client/tendermint/tx"
	"github.com/NPC-Chain/npcchub/client/utils"
	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
	"net/http"
	"strings"
)

const storeName = "stake"

func registerQueryRoutes(cliCtx context.CLIContext, r *mux.Router, cdc *codec.Codec) {

	// Get all delegations from a delegator
	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/delegations",
		delegatorDelegationsHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Get all unbonding delegations from a delegator
	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/unbonding-delegations",
		delegatorUnbondingDelegationsHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Get all redelegations from a delegator
	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/redelegations",
		delegatorRedelegationsHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Get all staking txs (i.e msgs) from a delegator
	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/txs",
		delegatorTxsHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Query all validators that a delegator is bonded to
	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/validators",
		delegatorValidatorsHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Query a validator that a delegator is bonded to
	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/validators/{validatorAddr}",
		delegatorValidatorHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Query a delegation between a delegator and a validator
	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/delegations/{validatorAddr}",
		delegationHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Query all unbonding delegations between a delegator and a validator
	r.HandleFunc(
		"/stake/delegators/{delegatorAddr}/unbonding-delegations/{validatorAddr}",
		unbondingDelegationHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Get all validators
	r.HandleFunc(
		"/stake/validators",
		validatorsHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Get a single validator info
	r.HandleFunc(
		"/stake/validators/{validatorAddr}",
		validatorHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Get all delegations to a validator
	r.HandleFunc(
		"/stake/validators/{validatorAddr}/delegations",
		validatorDelegationsHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Get all unbonding delegations from a validator
	r.HandleFunc(
		"/stake/validators/{validatorAddr}/unbonding-delegations",
		validatorUnbondingDelegationsHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Get all outgoing redelegations from a validator
	r.HandleFunc(
		"/stake/validators/{validatorAddr}/redelegations",
		validatorRedelegationsHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Get the current state of the staking pool
	r.HandleFunc(
		"/stake/pool",
		poolHandlerFn(cliCtx, cdc),
	).Methods("GET")

	// Get the current staking parameter values
	r.HandleFunc(
		"/stake/parameters",
		paramsHandlerFn(cliCtx, cdc),
	).Methods("GET")

}

// HTTP request handler to query a delegator delegations
func delegatorDelegationsHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryDelegator(cliCtx, cdc, "custom/stake/delegatorDelegations")
}

// HTTP request handler to query a delegator unbonding delegations
func delegatorUnbondingDelegationsHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryDelegator(cliCtx, cdc, "custom/stake/delegatorUnbondingDelegations")
}

// HTTP request handler to query a delegator redelegations
func delegatorRedelegationsHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryDelegator(cliCtx, cdc, "custom/stake/delegatorRedelegations")
}

// HTTP request handler to query all staking txs (msgs) from a delegator
func delegatorTxsHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var typesQuerySlice []string
		vars := mux.Vars(r)
		delegatorAddr := vars["delegatorAddr"]

		_, err := sdk.AccAddressFromBech32(delegatorAddr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		// Get values from query

		typesQuery := r.URL.Query().Get("type")
		trimmedQuery := strings.TrimSpace(typesQuery)
		if len(trimmedQuery) != 0 {
			typesQuerySlice = strings.Split(trimmedQuery, " ")
		}

		noQuery := len(typesQuerySlice) == 0
		isBondTx := contains(typesQuerySlice, "bond")
		isUnbondTx := contains(typesQuerySlice, "unbond")
		isRedTx := contains(typesQuerySlice, "redelegate")
		var txs = []tx.Info{}
		var actions []string

		switch {
		case isBondTx:
			actions = append(actions, string(tags.ActionDelegate))
		case isUnbondTx:
			actions = append(actions, string(tags.ActionBeginUnbonding))
			actions = append(actions, string(tags.ActionCompleteUnbonding))
		case isRedTx:
			actions = append(actions, string(tags.ActionBeginRedelegation))
			actions = append(actions, string(tags.ActionCompleteRedelegation))
		case noQuery:
			actions = append(actions, string(tags.ActionDelegate))
			actions = append(actions, string(tags.ActionBeginUnbonding))
			actions = append(actions, string(tags.ActionCompleteUnbonding))
			actions = append(actions, string(tags.ActionBeginRedelegation))
			actions = append(actions, string(tags.ActionCompleteRedelegation))
		default:
			w.WriteHeader(http.StatusNoContent)
			return
		}

		for _, action := range actions {
			foundTxs, errQuery := queryTxs(cliCtx, cdc, action, delegatorAddr)
			if errQuery != nil {
				utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			}
			txs = append(txs, foundTxs...)
		}

		utils.PostProcessResponse(w, cdc, txs, cliCtx.Indent)
	}
}

// HTTP request handler to query an unbonding-delegation
func unbondingDelegationHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryBonds(cliCtx, cdc, "custom/stake/unbondingDelegation")
}

// HTTP request handler to query a delegation
func delegationHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryBonds(cliCtx, cdc, "custom/stake/delegation")
}

// HTTP request handler to query all delegator bonded validators
func delegatorValidatorsHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryDelegator(cliCtx, cdc, "custom/stake/delegatorValidators")
}

// HTTP request handler to get information from a currently bonded validator
func delegatorValidatorHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryBonds(cliCtx, cdc, "custom/stake/delegatorValidator")
}

// HTTP request handler to query list of validators
func validatorsHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, sdk.AppendMsgToErr("could not parse query parameters", err.Error()))
			return
		}
		pageStr := r.FormValue("page")
		sizeStr := r.FormValue("size")
		params, err := ConvertPaginationParams(pageStr, sizeStr)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		bz, err := cdc.MarshalJSON(params)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}

		res, err := cliCtx.QueryWithData("custom/stake/validators", bz)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		var validators []types.Validator
		if err = cdc.UnmarshalJSON(res, &validators); err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		validatorOutputs := make([]stakeClient.ValidatorOutput, len(validators))
		for index, validator := range validators {
			validatorOutput := stakeClient.ConvertValidatorToValidatorOutput(cliCtx, validator)
			validatorOutputs[index] = validatorOutput
		}

		utils.PostProcessResponse(w, cdc, validatorOutputs, cliCtx.Indent)
	}
}

// HTTP request handler to query the validator information from a given validator address
func validatorHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryValidator(cliCtx, cdc, "custom/stake/validator")
}

// HTTP request handler to query all unbonding delegations from a validator
func validatorDelegationsHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryValidator(cliCtx, cdc, "custom/stake/validatorDelegations")
}

// HTTP request handler to query all unbonding delegations from a validator
func validatorUnbondingDelegationsHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryValidator(cliCtx, cdc, "custom/stake/validatorUnbondingDelegations")
}

// HTTP request handler to query all redelegations from a source validator
func validatorRedelegationsHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return queryValidator(cliCtx, cdc, "custom/stake/validatorRedelegations")
}

// HTTP request handler to query the pool information
func poolHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := cliCtx.QueryWithData("custom/stake/pool", nil)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		var poolStatus types.PoolStatus
		if err = cdc.UnmarshalJSON(res, &poolStatus); err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}

		poolOutput := stakeClient.ConvertPoolToPoolOutput(cliCtx, poolStatus)
		if res, err = codec.MarshalJSONIndent(cdc, poolOutput); err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		utils.PostProcessResponse(w, cdc, res, cliCtx.Indent)
	}
}

// HTTP request handler to query the staking params values
func paramsHandlerFn(cliCtx context.CLIContext, cdc *codec.Codec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := cliCtx.QueryWithData("custom/stake/parameters", nil)
		if err != nil {
			utils.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
		utils.PostProcessResponse(w, cdc, res, cliCtx.Indent)
	}
}
