package client

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

// nolint
const (
	// DefaultGasAdjustment is applied to gas estimates to avoid tx
	// execution failures due to state changes that might
	// occur between the tx simulation and the actual run.
	DefaultGasAdjustment = 1.5
	DefaultGasLimit      = 50000
	GasFlagSimulate      = "simulate"

	FlagUseLedger      = "ledger"
	FlagChainID        = "chain-id"
	FlagNode           = "node"
	FlagHeight         = "height"
	FlagGas            = "gas"
	FlagTrustNode      = "trust-node"
	FlagFrom           = "from"
	FlagFromAddr       = "from-addr"
	FlagAccountNumber  = "account-number"
	FlagSequence       = "sequence"
	FlagMemo           = "memo"
	FlagFee            = "fee"
	FlagAsync          = "async"
	FlagCommit         = "commit"
	FlagJson           = "json"
	FlagPrintResponse  = "print-response"
	FlagGenerateOnly   = "generate-only"
	FlagName           = "name"
	FlagIndentResponse = "indent"
	FlagDryRun         = "dry-run"
	FlagGasAdjustment  = "gas-adjustment"
)

// GetCommands adds common flags to query commands
func GetCommands(cmds ...*cobra.Command) []*cobra.Command {
	for _, c := range cmds {
		c.Flags().Bool(FlagIndentResponse, false, "Add indent to JSON response")
		c.Flags().Bool(FlagTrustNode, true, "Don't verify proofs for responses")
		c.Flags().Bool(FlagUseLedger, false, "Use a connected Ledger device")
		c.Flags().String(FlagChainID, "", "Chain ID of tendermint node")
		c.Flags().String(FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")
		c.Flags().Int64(FlagHeight, 0, "block height to query, omit to get most recent provable block")
	}
	return cmds
}

// PostCommands adds common flags for commands to post tx
func PostCommands(cmds ...*cobra.Command) []*cobra.Command {
	for _, c := range cmds {
		c.Flags().Bool(FlagIndentResponse, false, "Add indent to JSON response")
		c.Flags().String(FlagFrom, "", "Name of private key with which to sign")
		c.Flags().Uint64(FlagAccountNumber, 0, "AccountNumber number to sign the tx")
		c.Flags().Uint64(FlagSequence, 0, "Sequence number to sign the tx")
		c.Flags().String(FlagMemo, "", "Memo to send along with transaction")
		c.Flags().String(FlagFee, "", "Fee to pay along with transaction")
		c.Flags().String(FlagChainID, "", "Chain ID of tendermint node")
		c.Flags().String(FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")
		c.Flags().Bool(FlagUseLedger, false, "Use a connected Ledger device")
		c.Flags().Var(&GasFlagVar, FlagGas, fmt.Sprintf(
			"gas limit to set per-transaction; set to %q to calculate required gas automatically", GasFlagSimulate))
		c.Flags().Bool(FlagAsync, false, "broadcast transactions asynchronously")
		c.Flags().Bool(FlagCommit, false, "wait for transaction commit accomplishment, if true, --async will be ignored")
		c.Flags().Bool(FlagJson, false, "return output in json format")
		c.Flags().Bool(FlagTrustNode, true, "Don't verify proofs for responses")
		c.Flags().Bool(FlagPrintResponse, false, "return tx response (only works with async = false)")
		c.Flags().Bool(FlagGenerateOnly, false, "build an unsigned transaction and write it to STDOUT")
		c.Flags().String(FlagFromAddr, "", "Specify from address in generate-only mode")
		c.Flags().Bool(FlagDryRun, false, "ignore the --gas flag and perform a simulation of a transaction, but don't broadcast it")
		c.Flags().Float64(FlagGasAdjustment, DefaultGasAdjustment, "adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set manually this flag is ignored ")
	}
	return cmds
}

// LineBreak can be included in a command list to provide a blank line
// to help with readability
var (
	LineBreak  = &cobra.Command{Run: func(*cobra.Command, []string) {}}
	GasFlagVar = GasSetting{Gas: DefaultGasLimit}
)

// Gas flag parsing functions

// GasSetting encapsulates the possible values passed through the --gas flag.
type GasSetting struct {
	Simulate bool
	Gas      uint64
}

// Type returns the flag's value type.
func (v *GasSetting) Type() string { return "string" }

// Set parses and sets the value of the --gas flag.
func (v *GasSetting) Set(s string) (err error) {
	v.Simulate, v.Gas, err = ReadGasFlag(s)
	return
}

func (v *GasSetting) String() string {
	if v.Simulate {
		return GasFlagSimulate
	}
	return strconv.FormatUint(v.Gas, 10)
}

// ParseGasFlag parses the value of the --gas flag.
func ReadGasFlag(s string) (simulate bool, gas uint64, err error) {
	switch s {
	case "":
		gas = DefaultGasLimit
	case GasFlagSimulate:
		gas = 1
		simulate = true
	default:
		gas, err = strconv.ParseUint(s, 10, 64)
		if err != nil {
			err = fmt.Errorf("gas must be either integer or %q", GasFlagSimulate)
			return
		}
	}
	return
}
