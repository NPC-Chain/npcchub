package keys

import (
	"github.com/NPC-Chain/npcchub/client"
	"github.com/spf13/cobra"
)

// Commands registers a sub-tree of commands to interact with
// local private key storage.
func Commands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Add or view local private keys",
		Long: `Keys allows you to manage your local keystore for tendermint.

    These keys may be in any format supported by go-crypto and can be
    used by light-clients, full nodes, or any other application that
    needs to sign with a private key.`,
	}
	cmd.AddCommand(
		mnemonicKeyCommand(),
		newKeyCommand(),
		addKeyCommand(),
		exportKeyCommand(),
		listKeysCmd(),
		showKeysCmd(),
		client.LineBreak,
		deleteKeyCommand(),
		updateKeyCommand(),
	)
	return cmd
}
