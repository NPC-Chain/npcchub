package keys

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/NPC-Chain/npcchub/client"
	"github.com/NPC-Chain/npcchub/codec"
	"github.com/NPC-Chain/npcchub/crypto/keys"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/tendermint/tendermint/libs/cli"
	dbm "github.com/tendermint/tm-db"
)

// KeyDBName is the directory under root where we store the keys
const KeyDBName = "keys"

type BechKeyOutFn func(keyInfo keys.Info) (KeyOutput, error)

// keybase is used to make GetKeyBase a singleton
var keybase keys.Keybase

// TODO make keybase take a database not load from the directory

// initialize a keybase based on the configuration
func GetKeyBase() (keys.Keybase, error) {
	rootDir := viper.GetString(cli.HomeFlag)
	return GetKeyBaseFromDir(rootDir)
}

// GetKeyInfo returns key info for a given name. An error is returned if the
// keybase cannot be retrieved or getting the info fails.
func GetKeyInfo(name string) (keys.Info, error) {
	keybase, err := GetKeyBase()
	if err != nil {
		return nil, err
	}

	return keybase.Get(name)
}

// GetPassphrase returns a passphrase for a given name. It will first retrieve
// the key info for that name if the type is local, it'll fetch input from
// STDIN. Otherwise, an empty passphrase is returned. An error is returned if
// the key info cannot be fetched or reading from STDIN fails.
func GetPassphrase(name string) (string, error) {
	var passphrase string

	keyInfo, err := GetKeyInfo(name)
	if err != nil {
		return passphrase, err
	}

	// we only need a passphrase for locally stored keys
	// TODO: (ref: #864) address security concerns
	if keyInfo.GetType() == keys.TypeLocal {
		passphrase, err = ReadPassphraseFromStdin(name)
		if err != nil {
			return passphrase, err
		}
	}

	return passphrase, nil
}

// GetKeyBaseWithWritePerm initialize a keybase based on the configuration with write permissions.
func GetKeyBaseWithWritePerm() (keys.Keybase, error) {
	rootDir := viper.GetString(cli.HomeFlag)
	return GetKeyBaseFromDirWithWritePerm(rootDir)
}

// GetKeyBaseFromDirWithWritePerm initializes a keybase at a particular dir with write permissions.
func GetKeyBaseFromDirWithWritePerm(rootDir string) (keys.Keybase, error) {
	return getKeyBaseFromDirWithOpts(rootDir, nil)
}

func getKeyBaseFromDirWithOpts(rootDir string, o *opt.Options) (keys.Keybase, error) {
	if keybase == nil {
		db, err := dbm.NewGoLevelDBWithOpts(KeyDBName, filepath.Join(rootDir, "keys"), o)
		if err != nil {
			return nil, err
		}
		keybase = client.GetKeyBase(db)
	}
	return keybase, nil
}

// ReadPassphraseFromStdin attempts to read a passphrase from STDIN return an
// error upon failure.
func ReadPassphraseFromStdin(name string) (string, error) {
	buf := BufferStdin()
	prompt := fmt.Sprintf("Password to sign with '%s':", name)

	passphrase, err := GetPassword(prompt, buf)
	if err != nil {
		return passphrase, fmt.Errorf("Error reading passphrase: %v", err)
	}

	return passphrase, nil
}

func ReadKeystorePassphraseFromStdin() (string, error) {
	buf := BufferStdin()
	prompt1 := fmt.Sprintf("Enter the password for the keystore file signature:")
	prompt2 := fmt.Sprintf("Repeat the password:")

	passphrase, err := GetCheckPassword(prompt1, prompt2, buf)
	if err != nil {
		return passphrase, err
	}

	return passphrase, nil
}

// initialize a keybase based on the configuration
func GetKeyBaseFromDir(rootDir string) (keys.Keybase, error) {
	if keybase == nil {
		db, err := dbm.NewGoLevelDB(KeyDBName, filepath.Join(rootDir, "keys"))
		if err != nil {
			return nil, err
		}
		keybase = GetKeyBaseFromDB(db)
	}
	return keybase, nil
}

func GetKey(name string) (keys.Info, error) {
	kb, err := GetKeyBase()
	if err != nil {
		return nil, err
	}

	return kb.Get(name)
}

// used for outputting keys.Info over REST
type KeyOutput struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Address string `json:"address"`
	PubKey  string `json:"pub_key"`
	Seed    string `json:"seed,omitempty"`
}

// create a list of KeyOutput in bech32 format
func Bech32KeysOutput(infos []keys.Info) ([]KeyOutput, error) {
	kos := make([]KeyOutput, len(infos))
	for i, info := range infos {
		ko, err := Bech32KeyOutput(info)
		if err != nil {
			return nil, err
		}
		kos[i] = ko
	}
	return kos, nil
}

// Bech32KeyOutput create a KeyOutput in bech32 format
func Bech32KeyOutput(info keys.Info) (KeyOutput, error) {
	accAddr := sdk.AccAddress(info.GetPubKey().Address().Bytes())
	bechPubKey, err := sdk.Bech32ifyAccPub(info.GetPubKey())
	if err != nil {
		return KeyOutput{}, err
	}

	return KeyOutput{
		Name:    info.GetName(),
		Type:    info.GetType().String(),
		Address: accAddr.String(),
		PubKey:  bechPubKey,
	}, nil
}

// Bech32ConsKeyOutput returns key output for a consensus node's key
// information.
func Bech32ConsKeyOutput(keyInfo keys.Info) (KeyOutput, error) {
	consAddr := sdk.ConsAddress(keyInfo.GetPubKey().Address().Bytes())

	bechPubKey, err := sdk.Bech32ifyConsPub(keyInfo.GetPubKey())
	if err != nil {
		return KeyOutput{}, err
	}

	return KeyOutput{
		Name:    keyInfo.GetName(),
		Type:    keyInfo.GetType().String(),
		Address: consAddr.String(),
		PubKey:  bechPubKey,
	}, nil
}

// Bech32ValKeyOutput returns key output for a validator's key information.
func Bech32ValKeyOutput(keyInfo keys.Info) (KeyOutput, error) {
	valAddr := sdk.ValAddress(keyInfo.GetPubKey().Address().Bytes())

	bechPubKey, err := sdk.Bech32ifyValPub(keyInfo.GetPubKey())
	if err != nil {
		return KeyOutput{}, err
	}

	return KeyOutput{
		Name:    keyInfo.GetName(),
		Type:    keyInfo.GetType().String(),
		Address: valAddr.String(),
		PubKey:  bechPubKey,
	}, nil
}

func PrintKeyInfo(keyInfo keys.Info, bechKeyOut BechKeyOutFn) {
	ko, err := bechKeyOut(keyInfo)
	if err != nil {
		panic(err)
	}

	switch viper.Get(cli.OutputFlag) {
	case "text":
		fmt.Printf("NAME:\tTYPE:\tADDRESS:\t\t\t\t\t\tPUBKEY:\n")
		PrintKeyOutput(ko)
	case "json":
		var out []byte
		var err error

		if viper.GetBool(client.FlagIndentResponse) {
			out, err = cdc.MarshalJSONIndent(ko, "", "  ")
		} else {
			out, err = cdc.MarshalJSON(ko)
		}
		if err != nil {
			panic(err)
		}

		fmt.Println(string(out))
	}
}

func PrintInfos(cdc *codec.Codec, infos []keys.Info) {
	kos, err := Bech32KeysOutput(infos)
	if err != nil {
		panic(err)
	}
	switch viper.Get(cli.OutputFlag) {
	case "text":
		fmt.Printf("NAME:\tTYPE:\tADDRESS:\t\t\t\t\t\tPUBKEY:\n")
		for _, ko := range kos {
			PrintKeyOutput(ko)
		}
	case "json":
		var out []byte
		var err error

		if viper.GetBool(client.FlagIndentResponse) {
			out, err = cdc.MarshalJSONIndent(kos, "", "  ")
		} else {
			out, err = cdc.MarshalJSON(kos)
		}

		if err != nil {
			panic(err)
		}
		fmt.Println(string(out))
	}
}

func PrintKeyOutput(ko KeyOutput) {
	fmt.Printf("%s\t%s\t%s\t%s\n", ko.Name, ko.Type, ko.Address, ko.PubKey)
}

func PrintKeyAddress(info keys.Info, bechKeyOut BechKeyOutFn) {
	ko, err := bechKeyOut(info)
	if err != nil {
		panic(err)
	}

	fmt.Println(ko.Address)
}

func PrintPubKey(info keys.Info, bechKeyOut BechKeyOutFn) {
	ko, err := bechKeyOut(info)
	if err != nil {
		panic(err)
	}

	fmt.Println(ko.PubKey)
}

func GetBechKeyOut(bechPrefix string) (BechKeyOutFn, error) {
	switch bechPrefix {
	case "acc":
		return Bech32KeyOutput, nil
	case "val":
		return Bech32ValKeyOutput, nil
	case "cons":
		return Bech32ConsKeyOutput, nil
	}

	return nil, fmt.Errorf("invalid Bech32 prefix encoding provided: %s", bechPrefix)
}

// PostProcessResponse performs post process for rest response
func PostProcessResponse(w http.ResponseWriter, cdc *codec.Codec, response interface{}, indent bool) {
	var output []byte
	switch response.(type) {
	default:
		var err error
		if indent {
			output, err = cdc.MarshalJSONIndent(response, "", "  ")
		} else {
			output, err = cdc.MarshalJSON(response)
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
	case []byte:
		output = response.([]byte)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}
