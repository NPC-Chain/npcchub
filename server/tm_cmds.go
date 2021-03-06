package server

import (
	"fmt"

	"github.com/NPC-Chain/npcchub/codec"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/NPC-Chain/npcchub/client"
	sdk "github.com/NPC-Chain/npcchub/types"
	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/p2p"
)

// ShowNodeIDCmd - ported from Tendermint, dump node ID to stdout
func ShowNodeIDCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "show-node-id",
		Short: "Show this node's ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := ctx.Config
			nodeKey, err := p2p.LoadNodeKey(cfg.NodeKeyFile())
			if err != nil {
				return err
			}
			fmt.Println(nodeKey.ID())
			return nil
		},
	}
}

// ShowValidator - ported from Tendermint, show this node's validator info
func ShowValidatorCmd(ctx *Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "show-validator",
		Short: "Show this node's tendermint validator info",
		RunE: func(cmd *cobra.Command, args []string) error {

			cfg := ctx.Config
			privValidator, err := ReadPrivValidator(cfg.PrivValidatorFile())
			if err != nil {
				return err
			}
			valPubKey := privValidator.PubKey

			if viper.GetBool(client.FlagJson) {
				return printlnJSON(valPubKey)
			}

			pubkey, err := sdk.Bech32ifyConsPub(valPubKey)
			if err != nil {
				return err
			}

			fmt.Println(pubkey)
			return nil
		},
	}
	cmd.Flags().Bool(client.FlagJson, false, "get machine parseable output")
	return &cmd
}

// ShowAddressCmd - show this node's validator address
func ShowAddressCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-address",
		Short: "Shows this node's tendermint validator address",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := ctx.Config
			privValidator, err := ReadPrivValidator(cfg.PrivValidatorFile())
			if err != nil {
				return err
			}
			valConsAddr := (sdk.ConsAddress)(privValidator.Address)

			if viper.GetBool(client.FlagJson) {
				return printlnJSON(valConsAddr)
			}

			fmt.Println(valConsAddr.String())
			return nil
		},
	}

	cmd.Flags().Bool(client.FlagJson, false, "get machine parseable output")
	return cmd
}

func printlnJSON(v interface{}) error {
	cdc := codec.New()
	codec.RegisterCrypto(cdc)
	marshalled, err := cdc.MarshalJSON(v)
	if err != nil {
		return err
	}
	fmt.Println(string(marshalled))
	return nil
}

// UnsafeResetAllCmd - extension of the tendermint command, resets initialization
func UnsafeResetAllCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "unsafe-reset-all",
		Short: "Resets the blockchain database, removes address book files, and resets priv_validator.json to the genesis state",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := ctx.Config
			tcmd.ResetAll(cfg.DBDir(), cfg.P2P.AddrBookFile(), cfg.PrivValidatorFile(), ctx.Logger)
			return nil
		},
	}
}
