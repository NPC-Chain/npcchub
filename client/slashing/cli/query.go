package cli

import (
	"fmt"

	"github.com/NPC-Chain/npcchub/app/v1/slashing"
	"github.com/NPC-Chain/npcchub/client/context"
	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/spf13/cobra"
)

// GetCmdQuerySigningInfo implements the command to query signing info.
func GetCmdQuerySigningInfo(storeName string, cdc *codec.Codec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "signing-info [validator-pubkey]",
		Short:   "Query a validator's signing information",
		Example: "iriscli stake signing-info <validator public key>",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pk, err := sdk.GetConsPubKeyBech32(args[0])
			if err != nil {
				return err
			}

			key := slashing.GetValidatorSigningInfoKey(sdk.ConsAddress(pk.Address()))
			cliCtx := context.NewCLIContext().WithCodec(cdc)

			res, err := cliCtx.QueryStore(key, storeName)
			if err != nil {
				return err
			}
			if len(res) == 0 {
				return fmt.Errorf("the signing information of this validator %s is empty, please make sure its existence", args[0])
			}

			var signingInfo slashing.ValidatorSigningInfo
			err = cdc.UnmarshalBinaryLengthPrefixed(res, &signingInfo)
			if err != nil {
				return err
			}

			return cliCtx.PrintOutput(signingInfo)
		},
	}

	return cmd
}
