package server

import (
	"fmt"
	"io/ioutil"
	"path"

	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tmtypes "github.com/tendermint/tendermint/types"
)

const (
	flagHeight        = "height"
	flagForZeroHeight = "for-zero-height"
	flagOutputFile    = "output-file"
)

// ExportCmd dumps app state to JSON.
func ExportCmd(ctx *Context, cdc *codec.Codec, appExporter AppExporter) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export state to JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := viper.GetString("home")
			traceWriterFile := viper.GetString(flagTraceStore)
			emptyState, err := isEmptyState(home)
			if err != nil {
				return err
			}

			if emptyState {
				fmt.Println("WARNING: State is not initialized. Returning genesis file.")
				genesisFile := path.Join(home, "config", "genesis.json")
				genesis, err := ioutil.ReadFile(genesisFile)
				if err != nil {
					return err
				}
				fmt.Println(string(genesis))
				return nil
			}

			db, err := openDB(home)
			if err != nil {
				return err
			}
			traceWriter, err := openTraceWriter(traceWriterFile)
			if err != nil {
				return err
			}

			height := viper.GetInt64(flagHeight)
			if height < 0 {
				return errors.Errorf("Height must greater than or equal to zero")
			}
			forZeroHeight := viper.GetBool(flagForZeroHeight)
			exportHeight, appState, validators, err := appExporter(ctx, ctx.Logger, db, traceWriter, height, forZeroHeight)
			if err != nil {
				return errors.Errorf("error exporting state: %v\n", err)
			}

			doc, err := tmtypes.GenesisDocFromFile(ctx.Config.GenesisFile())
			if err != nil {
				return err
			}

			doc.AppState = appState
			doc.Validators = validators

			doc.AppState = sdk.MustSortJSON(doc.AppState)

			encoded, err := codec.MarshalJSONIndent(cdc, doc)
			if err != nil {
				return err
			}

			outputFile := viper.GetString(flagOutputFile)
			err = ioutil.WriteFile(outputFile, encoded, 0644)
			if err != nil {
				return err
			}
			fmt.Printf("export state from height %d to file %s successfully\n", exportHeight, outputFile)
			return nil
		},
	}
	cmd.Flags().Uint64(flagHeight, 0, "Export state from a particular height (0 means latest height)")
	cmd.Flags().Bool(flagForZeroHeight, false, "Export state to start at height zero (perform preproccessing)")
	cmd.Flags().String(flagOutputFile, "genesis.json", "Target file to save exported state")
	return cmd
}

func isEmptyState(home string) (bool, error) {
	files, err := ioutil.ReadDir(path.Join(home, "data"))
	if err != nil {
		return false, err
	}

	return len(files) == 0, nil
}
