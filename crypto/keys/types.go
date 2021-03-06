package keys

import (
	ccrypto "github.com/NPC-Chain/npcchub/crypto"
	"github.com/NPC-Chain/npcchub/crypto/keys/hd"
	"github.com/NPC-Chain/npcchub/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/multisig"
)

// Keybase exposes operations on a generic keystore
type Keybase interface {
	// CRUD on the keystore
	List() ([]Info, error)
	Get(name string) (Info, error)
	GetByAddress(address types.AccAddress) (Info, error)
	Delete(name, passphrase string, skipPass bool) error

	// Sign some bytes, looking up the private key to use
	Sign(name, passphrase string, msg []byte) ([]byte, crypto.PubKey, error)

	// CreateMnemonic creates a new mnemonic, and derives a hierarchical deterministic
	// key from that.
	CreateMnemonic(name string, language Language, passwd string, algo SigningAlgo) (info Info, seed string, err error)

	// CreateKey takes a mnemonic and derives, a password. This method is temporary
	CreateKey(name, mnemonic, passwd string) (info Info, err error)
	// CreateFundraiserKey takes a mnemonic and derives, a password
	CreateFundraiserKey(name, mnemonic, passwd string) (info Info, err error)
	// Compute a BIP39 seed from th mnemonic and bip39Passwd.
	// Derive private key from the seed using the BIP44 params.
	// Encrypt the key to disk using encryptPasswd.
	// See https://github.com/NPC-Chain/npcchub/issues/2095
	Derive(name, mnemonic, bip39Passwd,
		encryptPasswd string, params hd.BIP44Params) (Info, error)
	// Create, store, and return a new Ledger key reference
	CreateLedger(name string, path ccrypto.DerivationPath, algo SigningAlgo) (info Info, err error)

	// Create, store, and return a new offline key reference
	CreateOffline(name string, pubkey crypto.PubKey) (info Info, err error)

	// CreateMulti creates, stores, and returns a new multsig (offline) key reference
	CreateMulti(name string, pubkey crypto.PubKey) (info Info, err error)

	// The following operations will *only* work on locally-stored keys
	Update(name, oldpass string, getNewpass func() (string, error)) error
	Import(name string, armor string) (err error)
	ImportPubKey(name string, armor string) (err error)
	Export(name string) (armor string, err error)
	ExportPubKey(name string) (armor string, err error)

	// import from private key and store it
	ImportPrivateKey(name string, passwd string, privKey crypto.PrivKey) (info Info, err error)

	// *only* works on locally-stored keys. Temporary method until we redo the exporting API
	ExportPrivateKeyObject(name string, passphrase string) (crypto.PrivKey, error)

	// Close closes the database.
	CloseDB()
}

// KeyType reflects a human-readable type for key listing.
type KeyType uint

// Info KeyTypes
const (
	TypeLocal   KeyType = 0
	TypeLedger  KeyType = 1
	TypeOffline KeyType = 2
	TypeMulti   KeyType = 3
)

var keyTypes = map[KeyType]string{
	TypeLocal:   "local",
	TypeLedger:  "ledger",
	TypeOffline: "offline",
	TypeMulti:   "multi",
}

// String implements the stringer interface for KeyType.
func (kt KeyType) String() string {
	return keyTypes[kt]
}

// Info is the publicly exposed information about a keypair
type Info interface {
	// Human-readable type for key listing
	GetType() KeyType
	// Name of the key
	GetName() string
	// Public key
	GetPubKey() crypto.PubKey
	// Address
	GetAddress() types.AccAddress
}

var _ Info = &localInfo{}
var _ Info = &ledgerInfo{}
var _ Info = &offlineInfo{}

// localInfo is the public information about a locally stored key
type localInfo struct {
	Name         string        `json:"name"`
	PubKey       crypto.PubKey `json:"pubkey"`
	PrivKeyArmor string        `json:"privkey.armor"`
}

func newLocalInfo(name string, pub crypto.PubKey, privArmor string) Info {
	return &localInfo{
		Name:         name,
		PubKey:       pub,
		PrivKeyArmor: privArmor,
	}
}

func (i localInfo) GetType() KeyType {
	return TypeLocal
}

func (i localInfo) GetName() string {
	return i.Name
}

func (i localInfo) GetPubKey() crypto.PubKey {
	return i.PubKey
}

func (i localInfo) GetAddress() types.AccAddress {
	return i.PubKey.Address().Bytes()
}

// ledgerInfo is the public information about a Ledger key
type ledgerInfo struct {
	Name   string                 `json:"name"`
	PubKey crypto.PubKey          `json:"pubkey"`
	Path   ccrypto.DerivationPath `json:"path"`
}

func newLedgerInfo(name string, pub crypto.PubKey, path ccrypto.DerivationPath) Info {
	return &ledgerInfo{
		Name:   name,
		PubKey: pub,
		Path:   path,
	}
}

func (i ledgerInfo) GetType() KeyType {
	return TypeLedger
}

func (i ledgerInfo) GetName() string {
	return i.Name
}

func (i ledgerInfo) GetPubKey() crypto.PubKey {
	return i.PubKey
}

func (i ledgerInfo) GetAddress() types.AccAddress {
	return i.PubKey.Address().Bytes()
}

// offlineInfo is the public information about an offline key
type offlineInfo struct {
	Name   string        `json:"name"`
	PubKey crypto.PubKey `json:"pubkey"`
}

func newOfflineInfo(name string, pub crypto.PubKey) Info {
	return &offlineInfo{
		Name:   name,
		PubKey: pub,
	}
}

func (i offlineInfo) GetType() KeyType {
	return TypeOffline
}

func (i offlineInfo) GetName() string {
	return i.Name
}

func (i offlineInfo) GetPubKey() crypto.PubKey {
	return i.PubKey
}

func (i offlineInfo) GetAddress() types.AccAddress {
	return i.PubKey.Address().Bytes()
}

type multisigPubKeyInfo struct {
	PubKey crypto.PubKey `json:"pubkey"`
	Weight uint          `json:"weight"`
}

type multiInfo struct {
	Name      string               `json:"name"`
	PubKey    crypto.PubKey        `json:"pubkey"`
	Threshold uint                 `json:"threshold"`
	PubKeys   []multisigPubKeyInfo `json:"pubkeys"`
}

func NewMultiInfo(name string, pub crypto.PubKey) Info {
	multiPK := pub.(multisig.PubKeyMultisigThreshold)

	pubKeys := make([]multisigPubKeyInfo, len(multiPK.PubKeys))
	for i, pk := range multiPK.PubKeys {
		// TODO: Recursively check pk for total weight?
		pubKeys[i] = multisigPubKeyInfo{pk, 1}
	}

	return &multiInfo{
		Name:      name,
		PubKey:    pub,
		Threshold: multiPK.K,
		PubKeys:   pubKeys,
	}
}

func (i multiInfo) GetType() KeyType {
	return TypeMulti
}

func (i multiInfo) GetName() string {
	return i.Name
}

func (i multiInfo) GetPubKey() crypto.PubKey {
	return i.PubKey
}

func (i multiInfo) GetAddress() types.AccAddress {
	return i.PubKey.Address().Bytes()
}

// encoding info
func writeInfo(i Info) []byte {
	return cdc.MustMarshalBinaryLengthPrefixed(i)
}

// decoding info
func readInfo(bz []byte) (info Info, err error) {
	err = cdc.UnmarshalBinaryLengthPrefixed(bz, &info)
	return
}
