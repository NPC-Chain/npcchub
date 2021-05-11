package keys

import (
	"github.com/NPC-Chain/npcchub/crypto/keys"
	dbm "github.com/tendermint/tm-db"
)

// GetKeyBase initializes a keybase based on the given db.
// The KeyBase manages all activity requiring access to a key.
func GetKeyBaseFromDB(db dbm.DB) keys.Keybase {
	keybase := keys.New(
		db,
	)
	return keybase
}

// MockKeyBase generates an in-memory keybase that will be discarded
// useful for --dry-run to generate a seed phrase without
// storing the key
func MockKeyBase() keys.Keybase {
	return GetKeyBaseFromDB(dbm.NewMemDB())
}
