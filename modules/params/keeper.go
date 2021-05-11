package params

import (
	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"

	"github.com/NPC-Chain/npcchub/modules/params/subspace"
)

// Keeper of the global paramstore
type Keeper struct {
	cdc  *codec.Codec
	key  sdk.StoreKey
	tkey sdk.StoreKey

	spaces    map[string]*Subspace
	paramSets map[string]ParamSet
}

// NewKeeper constructs a params keeper
func NewKeeper(cdc *codec.Codec, key *sdk.KVStoreKey, tkey *sdk.TransientStoreKey) (k Keeper) {
	k = Keeper{
		cdc:  cdc,
		key:  key,
		tkey: tkey,

		spaces:    make(map[string]*Subspace),
		paramSets: make(map[string]ParamSet),
	}

	return k
}

// Allocate subspace used for keepers
func (k Keeper) Subspace(spacename string) Subspace {
	_, ok := k.spaces[spacename]
	if ok {
		panic("subspace already occupied")
	}

	if spacename == "" {
		panic("cannot use empty string for subspace")
	}

	space := subspace.NewSubspace(k.cdc, k.key, k.tkey, spacename)

	k.spaces[spacename] = &space

	return space
}

// Get existing substore from keeper
func (k Keeper) GetSubspace(storename string) (Subspace, bool) {
	space, ok := k.spaces[storename]
	if !ok {
		return Subspace{}, false
	}
	return *space, ok
}

func (k Keeper) RegisterParamSet(ps ...ParamSet) {
	RegisterParamSet(k.paramSets, ps...)
}

// Get existing substore from keeper
func (k Keeper) GetParamSet(paramSpace string) (ParamSet, bool) {
	paramSet, ok := k.paramSets[paramSpace]
	if !ok {
		return nil, false
	}
	return paramSet, ok
}
