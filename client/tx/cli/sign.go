package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/NPC-Chain/npcchub/app/v1/auth"
	"github.com/NPC-Chain/npcchub/client"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/client/utils"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/multisig"
)

const (
	flagMultisig     = "multisig"
	flagAppend       = "append"
	flagValidateSigs = "validate-signatures"
	flagOffline      = "offline"
	flagSigOnly      = "signature-only"
	flagOutfile      = "output-document"
)

// GetSignCommand returns the sign command
func GetSignCommand(codec *amino.Codec, decoder auth.AccountDecoder) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sign <file>",
		Short: "Sign transactions generated offline",
		Long: `Sign transactions created with the --generate-only flag.
Read a transaction from <file>, sign it, and print its JSON encoding.

If the flag --signature-only flag is set, it will output a JSON representation
of the generated signature only.

The --offline flag makes sure that the client will not reach out to the local cache.
Thus account number or sequence number lookups will not be performed and it is
recommended to set such parameters manually.

The --multisig=<multisig_key> flag generates a signature on behalf of a multisig account
key. It implies --signature-only. Full multisig signed transactions may eventually
be generated via the 'multisign' command.`,
		Example: "iriscli tx sign <file> --name <key name> --chain-id=<chain-id>",
		PreRun:  preSignCmd,
		RunE:    makeSignCmd(codec, decoder),
		Args:    cobra.ExactArgs(1),
	}

	cmd.Flags().String(
		flagMultisig, "",
		"Address of the multisig account on behalf of which the transaction shall be signed",
	)
	cmd.Flags().Bool(
		flagAppend, true,
		"Append the signature to the existing ones. If disabled, old signatures would be overwritten. Ignored if --multisig is on",
	)
	cmd.Flags().Bool(
		flagValidateSigs, false,
		"Print the addresses that must sign the transaction, those who have already signed it, and make sure that signatures are in the correct order",
	)

	cmd.Flags().String(client.FlagName, "", "Name of private key with which to sign")
	cmd.Flags().Bool(flagSigOnly, false, "Print only the generated signature, then exit")
	cmd.Flags().Bool(flagOffline, false, "Offline mode. Do not query local cache.")
	cmd.Flags().String(flagOutfile, "", "The document will be written to the given file instead of STDOUT")
	return cmd
}

func preSignCmd(cmd *cobra.Command, _ []string) {
	// Conditionally mark the account and sequence numbers required as no RPC
	// query will be done.
	if viper.GetBool(flagOffline) {
		cmd.MarkFlagRequired(client.FlagAccountNumber)
		cmd.MarkFlagRequired(client.FlagSequence)
	}
}

func makeSignCmd(cdc *amino.Codec, decoder auth.AccountDecoder) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		if len(viper.GetString(client.FlagChainID)) == 0 {
			return fmt.Errorf("missing chain-id")
		}
		stdTx, err := readAndUnmarshalStdTx(cdc, args[0])
		if err != nil {
			return
		}

		offline := viper.GetBool(flagOffline)
		name := viper.GetString(client.FlagName)
		cliCtx := context.NewCLIContext().WithCodec(cdc).WithAccountDecoder(decoder)
		txCtx := utils.NewTxContextFromCLI()

		if viper.GetBool(flagValidateSigs) {
			if !printAndValidateSigs(cliCtx, txCtx.ChainID, stdTx, offline) {
				return fmt.Errorf("signatures validation failed")
			}

			return nil
		}

		// if --signature-only is on, then override --append
		var newTx auth.StdTx
		generateSignatureOnly := viper.GetBool(flagSigOnly)
		multisigAddrStr := viper.GetString(flagMultisig)

		if multisigAddrStr != "" {
			var multisigAddr sdk.AccAddress

			multisigAddr, err = sdk.AccAddressFromBech32(multisigAddrStr)
			if err != nil {
				return err
			}
			newTx, err = utils.SignStdTxWithSignerAddress(
				txCtx, cliCtx, multisigAddr, name, stdTx, offline,
			)

			if err != nil {
				return err
			}
			generateSignatureOnly = true
		} else {
			appendSig := viper.GetBool(flagAppend) && !generateSignatureOnly
			newTx, err = utils.SignStdTx(txCtx, cliCtx, name, stdTx, appendSig, offline)
		}

		if err != nil {
			return err
		}

		var json []byte

		switch generateSignatureOnly {
		case true:
			switch cliCtx.Indent {
			case true:
				json, err = cdc.MarshalJSONIndent(newTx.Signatures[0], "", "  ")

			default:
				json, err = cdc.MarshalJSON(newTx.Signatures[0])
			}

		default:
			switch cliCtx.Indent {
			case true:
				json, err = cdc.MarshalJSONIndent(newTx, "", "  ")

			default:
				json, err = cdc.MarshalJSON(newTx)
			}
		}

		if err != nil {
			return err
		}

		if viper.GetString(flagOutfile) == "" {
			fmt.Printf("%s\n", json)
			return
		}

		fp, err := os.OpenFile(
			viper.GetString(flagOutfile), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644,
		)
		if err != nil {
			return err
		}

		defer fp.Close()
		fmt.Fprintf(fp, "%s\n", json)

		return
	}
}

func readAndUnmarshalStdTx(cdc *amino.Codec, filename string) (stdTx auth.StdTx, err error) {
	var bytes []byte
	if bytes, err = ioutil.ReadFile(filename); err != nil {
		return
	}
	if err = cdc.UnmarshalJSON(bytes, &stdTx); err != nil {
		return
	}
	return
}

// printAndValidateSigs will validate the signatures of a given transaction over
// its expected signers. In addition, if offline has not been supplied, the
// signature is verified over the transaction sign bytes.
func printAndValidateSigs(
	cliCtx context.CLIContext, chainID string, stdTx auth.StdTx, offline bool,
) bool {

	fmt.Println("Signers:")

	signers := stdTx.GetSigners()
	for i, signer := range signers {
		fmt.Printf("  %v: %v\n", i, signer.String())
	}

	success := true
	sigs := stdTx.GetSignatures()

	fmt.Println("")
	fmt.Println("Signatures:")

	if len(sigs) != len(signers) {
		success = false
	}

	for i, sig := range sigs {
		sigAddr := sdk.AccAddress(sig.Address())
		sigSanity := "OK"

		var (
			multiSigHeader string
			multiSigMsg    string
		)

		if i >= len(signers) || !sigAddr.Equals(signers[i]) {
			sigSanity = "ERROR: signature does not match its respective signer"
			success = false
		}

		// Validate the actual signature over the transaction bytes since we can
		// reach out to a full node to query accounts.
		if !offline && success {
			acc, err := cliCtx.GetAccount(sigAddr)
			if err != nil {
				fmt.Printf("failed to get account: %s\n", sigAddr)
				return false
			}

			sigBytes := auth.StdSignBytes(
				chainID, acc.GetAccountNumber(), acc.GetSequence(),
				stdTx.Fee, stdTx.GetMsgs(), stdTx.GetMemo(),
			)

			if ok := sig.VerifyBytes(sigBytes, sig.Signature); !ok {
				sigSanity = "ERROR: signature invalid"
				success = false
			}
		}

		multiPK, ok := sig.PubKey.(multisig.PubKeyMultisigThreshold)
		if ok {
			var multiSig multisig.Multisignature
			cliCtx.Codec.MustUnmarshalBinaryBare(sig.Signature, &multiSig)

			var b strings.Builder
			b.WriteString("\n  MultiSig Signatures:\n")

			for i := 0; i < multiSig.BitArray.Size(); i++ {
				if multiSig.BitArray.GetIndex(i) {
					addr := sdk.AccAddress(multiPK.PubKeys[i].Address().Bytes())
					b.WriteString(fmt.Sprintf("    %d: %s (weight: %d)\n", i, addr, 1))
				}
			}

			multiSigHeader = fmt.Sprintf(" [multisig threshold: %d/%d]", multiPK.K, len(multiPK.PubKeys))
			multiSigMsg = b.String()
		}

		fmt.Printf("  %d: %s\t\t\t[%s]%s%s\n", i, sigAddr.String(), sigSanity, multiSigHeader, multiSigMsg)
	}

	fmt.Println("")
	return success
}
