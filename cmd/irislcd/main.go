package main

import (
	"github.com/NPC-Chain/npcchub/app"
	"github.com/NPC-Chain/npcchub/lite"
	_ "github.com/NPC-Chain/npcchub/lite/statik"
	"github.com/NPC-Chain/npcchub/version"
	"github.com/spf13/cobra"
	"github.com/tendermint/tendermint/libs/cli"
)

// rootCmd is the entry point for this binary
var (
	rootCmd = &cobra.Command{
		Use:   "irislcd",
		Short: "IRIS Hub API Server (Lite Client Daemon)",
	}
)

func main() {
	// sdk.InitBech32Prefix()
	cobra.EnableCommandSorting = false
	cdc := app.MakeLatestCodec()

	rootCmd.AddCommand(
		lite.ServeLCDStartCommand(cdc),
		version.ServeVersionCommand(cdc),
	)

	// prepare and add flags
	executor := cli.PrepareMainCmd(rootCmd, "IRISLCD", app.DefaultLCDHome)
	err := executor.Execute()
	if err != nil {
		// handle with #870
		panic(err)
	}
}
