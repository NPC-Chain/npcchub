package main

import (
	"os"

	"github.com/NPC-Chain/npcchub/app"
	debugcmd "github.com/NPC-Chain/npcchub/tools/debug"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/cli"
)

func init() {
	//	sdk.InitBech32Prefix()
	rootCmd.AddCommand(debugcmd.RootCmd)
}

var rootCmd = &cobra.Command{
	Use:          "iristool",
	Short:        "Iris tool",
	SilenceUsage: true,
}

func main() {
	executor := cli.PrepareMainCmd(rootCmd, "IRIS", app.DefaultNodeHome)
	err := executor.Execute()
	if err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
