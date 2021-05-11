package utils

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/NPC-Chain/npcchub/app/v1/auth"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
)

const (
	Async                = "async"
	Commit               = "commit"
	queryArgDryRun       = "simulate"
	queryArgGenerateOnly = "generate-only"
)

//----------------------------------------
// Basic HTTP utilities

// WriteErrorResponse prepares and writes a HTTP error
// given a status code and an error message.
func WriteErrorResponse(w http.ResponseWriter, status int, err string) {
	w.WriteHeader(status)
	w.Write([]byte(err))
}

// WriteSimulationResponse prepares and writes an HTTP
// response for transactions simulations.
type kvPair struct {
	TagKey   string `json:"tag_key"`
	TagValue string `json:"tag_value"`
}
type abciResult struct {
	Code      sdk.CodeType `json:"code"`
	Data      []byte       `json:"data"`
	Log       string       `json:"log"`
	GasWanted uint64       `json:"gas_wanted"`
	GasUsed   uint64       `json:"gas_used"`
	FeeAmount int64        `json:"fee_amount"`
	FeeDenom  string       `json:"fee_denom"`
	Tags      []kvPair     `json:"tags"`
}
type simulateResult struct {
	GasEstimate uint64     `json:"gas_estimate"`
	Result      abciResult `json:"result"`
}

func WriteSimulationResponse(w http.ResponseWriter, cliCtx context.CLIContext, gas uint64, result sdk.Result) {
	w.WriteHeader(http.StatusOK)
	var kvPairs []kvPair
	for _, tag := range result.Tags {
		kvPairs = append(kvPairs, kvPair{
			TagKey:   string(tag.Key),
			TagValue: string(tag.Value),
		})
	}
	abciResult := abciResult{
		Code:      result.Code,
		Data:      result.Data,
		Log:       result.Log,
		GasWanted: result.GasWanted,
		GasUsed:   result.GasUsed,
		FeeAmount: result.FeeAmount,
		FeeDenom:  result.FeeDenom,
		Tags:      kvPairs,
	}

	simulateResult := simulateResult{
		GasEstimate: gas,
		Result:      abciResult,
	}
	var output []byte
	var err error
	if cliCtx.Indent {
		output, err = cliCtx.Codec.MarshalJSONIndent(simulateResult, "", "  ")
	} else {
		output, err = cliCtx.Codec.MarshalJSON(simulateResult)
	}
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Write(output)
}

// HasDryRunArg returns true if the request's URL query contains the dry run
// argument and its value is set to "true".
func HasDryRunArg(r *http.Request) bool {
	return urlQueryHasArg(r.URL, queryArgDryRun)
}

// AsyncOnlyArg returns whether a URL's query "async" parameter
func AsyncOnlyArg(r *http.Request) bool {
	return urlQueryHasArg(r.URL, Async)
}

// CommitOnlyArg returns whether a URL's query "commit" parameter
func CommitOnlyArg(r *http.Request) bool {
	return urlQueryHasArg(r.URL, Commit)
}

// ParseInt64OrReturnBadRequest converts s to a int64 value.
func ParseInt64OrReturnBadRequest(w http.ResponseWriter, s string) (n int64, ok bool) {
	var err error

	n, err = strconv.ParseInt(s, 10, 64)
	if err != nil {
		err := fmt.Errorf("'%s' is not a valid int64", s)
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return n, false
	}

	return n, true
}

// ParseUint64OrReturnBadRequest converts s to a uint64 value.
func ParseUint64OrReturnBadRequest(w http.ResponseWriter, s string) (n uint64, ok bool) {
	var err error
	n, err = strconv.ParseUint(s, 10, 64)
	if err != nil {
		err := fmt.Errorf("'%s' is not a valid uint64", s)
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return n, false
	}
	return n, true
}

// ParseFloat64OrReturnBadRequest converts s to a float64 value. It returns a
// default value, defaultIfEmpty, if the string is empty.
func ParseFloat64OrReturnBadRequest(w http.ResponseWriter, s string, defaultIfEmpty float64) (n float64, ok bool) {
	if len(s) == 0 {
		return defaultIfEmpty, true
	}

	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return n, false
	}

	return n, true
}

// WriteGenerateStdTxResponse writes response for the generate_only mode.
func WriteGenerateStdTxResponse(w http.ResponseWriter, txCtx TxContext, msgs []sdk.Msg) {
	stdMsg, err := txCtx.Build(msgs)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	output, err := txCtx.Codec.MarshalJSON(auth.NewStdTx(stdMsg.Msgs, stdMsg.Fee, nil, stdMsg.Memo))
	if err != nil {
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Write(output)
	return
}

func urlQueryHasArg(url *url.URL, arg string) bool { return url.Query().Get(arg) == "true" }

// ReadPostBody
func ReadPostBody(w http.ResponseWriter, r *http.Request, cdc *codec.Codec, req interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid post body")
			WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		}
	}()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return err
	}

	err = cdc.UnmarshalJSON(body, req)
	if err != nil {
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return err
	}

	return nil
}

// InitReqCliCtx
func InitReqCliCtx(cliCtx context.CLIContext, r *http.Request) context.CLIContext {
	cliCtx.Async = AsyncOnlyArg(r)
	cliCtx.DryRun = HasDryRunArg(r)
	cliCtx.Commit = CommitOnlyArg(r)
	return cliCtx
}

// BuildReqTxCtx builds a tx context for the request.
// Make sure baseTx has been validated
func BuildReqTxCtx(cliCtx context.CLIContext, baseTx BaseTx, w http.ResponseWriter) TxContext {
	gas, _ := strconv.ParseUint(baseTx.Gas, 10, 64)

	txCtx := TxContext{
		ChainID: baseTx.ChainID,
		Gas:     gas,
		Fee:     baseTx.Fee,
		Memo:    baseTx.Memo,
	}

	txCtx = txCtx.WithCodec(cliCtx.Codec)

	return txCtx
}

// PostProcessResponse performs post process for rest response
func PostProcessResponse(w http.ResponseWriter, cdc *codec.Codec, response interface{}, indent bool) {
	var output []byte
	switch response.(type) {
	default:
		var err error
		if indent {
			output, err = cdc.MarshalJSONIndent(response, "", "  ")
		} else {
			output, err = cdc.MarshalJSON(response)
		}
		if err != nil {
			WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
			return
		}
	case []byte:
		output = response.([]byte)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}
