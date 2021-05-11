package keys

import (
	"fmt"
	"github.com/NPC-Chain/npcchub/client/keys"
	"github.com/spf13/cobra"
)

func updateKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update <name>",
		Short:   "Change the password used to protect private key",
		Example: "iriscli keys update <key name>",
		RunE:    runUpdateCmd,
		Args:    cobra.ExactArgs(1),
	}
	return cmd
}

func runUpdateCmd(cmd *cobra.Command, args []string) error {
	name := args[0]

	buf := keys.BufferStdin()
	kb, err := keys.GetKeyBaseWithWritePerm()
	if err != nil {
		return err
	}
	oldpass, err := keys.GetPassword(
		"Enter the current passphrase:", buf)
	if err != nil {
		return err
	}

	getNewpass := func() (string, error) {
		return keys.GetCheckPassword(
			"Enter the new passphrase:",
			"Repeat the new passphrase:", buf)
	}

	err = kb.Update(name, oldpass, getNewpass)
	if err != nil {
		return err
	}
	fmt.Println("Password successfully updated!")
	return nil
}
