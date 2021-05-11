package upgrade

import (
	"encoding/hex"
	"github.com/NPC-Chain/npcchub/codec"
	"github.com/NPC-Chain/npcchub/modules/auth"
	"github.com/NPC-Chain/npcchub/modules/bank"
	"github.com/NPC-Chain/npcchub/modules/params"
	"github.com/NPC-Chain/npcchub/modules/stake"
	"github.com/NPC-Chain/npcchub/store"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
	"os"
	"testing"
)

var (
	pks = []crypto.PubKey{
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB50"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB51"),
		newPubKey("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AFB52"),
	}
	addrs = []sdk.AccAddress{
		sdk.AccAddress(pks[0].Address()),
		sdk.AccAddress(pks[1].Address()),
		sdk.AccAddress(pks[2].Address()),
	}
	initCoins sdk.Int = sdk.NewInt(200)
)

func newPubKey(pk string) (res crypto.PubKey) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		panic(err)
	}
	var pkEd ed25519.PubKeyEd25519
	copy(pkEd[:], pkBytes[:])
	return pkEd
}

func createTestCodec() *codec.Codec {
	cdc := codec.New()
	sdk.RegisterCodec(cdc)
	RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	bank.RegisterCodec(cdc)
	stake.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	return cdc
}

func createTestInput(t *testing.T) (sdk.Context, Keeper, params.Keeper) {
	keyMain := sdk.NewKVStoreKey("main")
	keyAcc := sdk.NewKVStoreKey("acc")
	keyStake := sdk.NewKVStoreKey("stake")
	keyUpgrade := sdk.NewKVStoreKey("upgrade")
	keyParams := sdk.NewKVStoreKey("params")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	tkeyParams := sdk.NewTransientStoreKey("transient_params")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyMain, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStake, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyUpgrade, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyStake, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)

	err := ms.LoadLatestVersion()
	require.Nil(t, err)
	ctx := sdk.NewContext(ms, abci.Header{}, false, log.NewTMLogger(os.Stdout))
	cdc := createTestCodec()
	AccountKeeper := auth.NewAccountKeeper(cdc, keyAcc, auth.ProtoBaseAccount)
	ck := bank.NewBaseKeeper(AccountKeeper)

	paramsKeeper := params.NewKeeper(
		cdc,
		keyParams, tkeyParams,
	)
	sk := stake.NewKeeper(
		cdc,
		keyStake, tkeyStake,
		ck, paramsKeeper.Subspace(stake.DefaultParamspace),
		stake.DefaultCodespace,
		stake.NopMetrics(),
	)
	keeper := NewKeeper(cdc, keyUpgrade, sdk.NewProtocolKeeper(sdk.NewKVStoreKey("main")), sk, NopMetrics())

	return ctx, keeper, paramsKeeper
}
