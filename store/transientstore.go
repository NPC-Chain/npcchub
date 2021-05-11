package store

import (
	sdk "github.com/NPC-Chain/npcchub/types"
	dbm "github.com/tendermint/tm-db"
)

var _ KVStore = (*transientStore)(nil)

// transientStore is a wrapper for a MemDB with Commiter implementation
type transientStore struct {
	dbStoreAdapter
}

// Constructs new MemDB adapter
func newTransientStore() *transientStore {
	return &transientStore{dbStoreAdapter{dbm.NewMemDB()}}
}

// Implements CommitStore
// Commit cleans up transientStore.
func (ts *transientStore) Commit([]*sdk.KVStoreKey) (id CommitID) {
	ts.dbStoreAdapter = dbStoreAdapter{dbm.NewMemDB()}
	return
}

// Commit cleans up transientStore.
func (ts *transientStore) CommitWithVersion([]*sdk.KVStoreKey, int64) (id CommitID) {
	ts.dbStoreAdapter = dbStoreAdapter{dbm.NewMemDB()}
	return
}

// Implements CommitStore
func (ts *transientStore) SetPruning(pruning PruningStrategy) {
}

// Implements CommitStore
func (ts *transientStore) LastCommitID() (id CommitID) {
	return
}

// Implements KVStore
func (ts *transientStore) Prefix(prefix []byte) KVStore {
	return prefixStore{ts, prefix}
}

// Implements KVStore
func (ts *transientStore) Gas(meter GasMeter, config GasConfig) KVStore {
	return NewGasKVStore(meter, config, ts)
}

// Implements Store.
func (ts *transientStore) GetStoreType() StoreType {
	return sdk.StoreTypeTransient
}
