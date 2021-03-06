package init

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/NPC-Chain/npcchub/app/v2"
	"github.com/NPC-Chain/npcchub/codec"
	"github.com/NPC-Chain/npcchub/server"
	"github.com/tendermint/go-amino"
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"
)

// ExportGenesisFile creates and writes the genesis configuration to disk. An
// error is returned if building or writing the configuration to file fails.
func ExportGenesisFile(
	genFile, chainID string, validators []types.GenesisValidator, appState json.RawMessage,
) error {

	genDoc := types.GenesisDoc{
		ChainID:    chainID,
		Validators: validators,
		AppState:   appState,
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}

	return genDoc.SaveAs(genFile)
}

// ExportGenesisFileWithTime creates and writes the genesis configuration to disk.
// An error is returned if building or writing the configuration to file fails.
func ExportGenesisFileWithTime(
	genFile, chainID string, validators []types.GenesisValidator,
	appState json.RawMessage, genTime time.Time,
) error {

	genDoc := types.GenesisDoc{
		GenesisTime: genTime,
		ChainID:     chainID,
		Validators:  validators,
		AppState:    appState,
	}

	if err := genDoc.ValidateAndComplete(); err != nil {
		return err
	}

	return genDoc.SaveAs(genFile)
}

// read of create the private key file for this config
func ReadOrCreatePrivValidator(privValFile string) crypto.PubKey {
	var privValidator *privval.FilePV

	if common.FileExists(privValFile) {
		privValidator = privval.LoadFilePV(privValFile)
	} else {
		privValidator = privval.GenFilePV(privValFile)
		privValidator.Save()
	}

	return privValidator.GetPubKey()
}

// InitializeNodeValidatorFiles creates private validator and p2p configuration files.
func InitializeNodeValidatorFiles(
	config *cfg.Config) (nodeID string, valPubKey crypto.PubKey, err error,
) {

	nodeKey, err := p2p.LoadOrGenNodeKey(config.NodeKeyFile())
	if err != nil {
		return nodeID, valPubKey, err
	}

	nodeID = string(nodeKey.ID())
	valPubKey = ReadOrCreatePrivValidator(config.PrivValidatorFile())

	return nodeID, valPubKey, nil
}

func LoadNodeValidatorFiles(
	config *cfg.Config) (nodeID string, valPubKey crypto.PubKey, err error,
) {

	nodeKey, err := p2p.LoadNodeKey(config.NodeKeyFile())
	if err != nil {
		return nodeID, valPubKey, err
	}

	nodeID = string(nodeKey.ID())
	pv, err := server.ReadPrivValidator(config.PrivValidatorFile())
	if err != nil {
		return nodeID, valPubKey, err
	}
	valPubKey = pv.GetPubKey()

	return nodeID, valPubKey, err
}

func loadGenesisDoc(cdc *amino.Codec, genFile string) (genDoc types.GenesisDoc, err error) {
	genContents, err := ioutil.ReadFile(genFile)
	if err != nil {
		return genDoc, err
	}

	if err := cdc.UnmarshalJSON(genContents, &genDoc); err != nil {
		return genDoc, err
	}

	return genDoc, err
}

func initializeEmptyGenesis(
	cdc *codec.Codec, genFile, chainID string, overwrite bool,
) (appState json.RawMessage, err error) {

	if !overwrite && common.FileExists(genFile) {
		return nil, fmt.Errorf("genesis.json file already exists: %v", genFile)
	}

	return codec.MarshalJSONIndent(cdc, v2.NewDefaultGenesisFileState())
}
