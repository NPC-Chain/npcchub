package keeper

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"strconv"
	"testing"

	"github.com/NPC-Chain/npcchub/codec"
	"github.com/NPC-Chain/npcchub/modules/auth"
	"github.com/NPC-Chain/npcchub/modules/bank"
	"github.com/NPC-Chain/npcchub/modules/params"
	"github.com/NPC-Chain/npcchub/modules/stake/types"
	"github.com/NPC-Chain/npcchub/store"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/libs/log"
	tmtypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"
)

// dummy addresses used for testing
var (
	Addrs       = createTestAddrs(500)
	PKs         = createTestPubKeys(500)
	emptyAddr   sdk.AccAddress
	emptyPubkey crypto.PubKey

	addrDels = []sdk.AccAddress{
		Addrs[0],
		Addrs[1],
	}
	addrVals = []sdk.ValAddress{
		sdk.ValAddress(Addrs[2]),
		sdk.ValAddress(Addrs[3]),
		sdk.ValAddress(Addrs[4]),
		sdk.ValAddress(Addrs[5]),
		sdk.ValAddress(Addrs[6]),
	}
)

//_______________________________________________________________________________________

// intended to be used with require/assert:  require.True(ValEq(...))
func ValEq(t *testing.T, exp, got types.Validator) (*testing.T, bool, string, types.Validator, types.Validator) {
	return t, exp.Equal(got), "expected:\t%v\ngot:\t\t%v", exp, got
}

//_______________________________________________________________________________________

// create a codec used only for testing
func MakeTestCodec() *codec.Codec {
	var cdc = codec.New()

	// Register Msgs
	cdc.RegisterInterface((*sdk.Msg)(nil), nil)
	cdc.RegisterConcrete(bank.MsgSend{}, "test/stake/Send", nil)
	cdc.RegisterConcrete(bank.MsgIssue{}, "test/stake/Issue", nil)
	cdc.RegisterConcrete(types.MsgCreateValidator{}, "test/stake/CreateValidator", nil)
	cdc.RegisterConcrete(types.MsgEditValidator{}, "test/stake/EditValidator", nil)
	cdc.RegisterConcrete(types.MsgBeginUnbonding{}, "test/stake/BeginUnbonding", nil)
	cdc.RegisterConcrete(types.MsgBeginRedelegate{}, "test/stake/BeginRedelegate", nil)

	// Register AppAccount
	cdc.RegisterInterface((*auth.Account)(nil), nil)
	cdc.RegisterConcrete(&auth.BaseAccount{}, "test/stake/Account", nil)
	codec.RegisterCrypto(cdc)

	return cdc
}

// hogpodge of all sorts of input required for testing
func CreateTestInput(t *testing.T, isCheckTx bool, initCoins sdk.Int) (sdk.Context, auth.AccountKeeper, Keeper) {

	keyStake := sdk.NewKVStoreKey("stake")
	tkeyStake := sdk.NewTransientStoreKey("transient_stake")
	keyAcc := sdk.NewKVStoreKey("acc")
	keyParams := sdk.NewKVStoreKey("params")
	tkeyParams := sdk.NewTransientStoreKey("transient_params")

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(tkeyStake, sdk.StoreTypeTransient, nil)
	ms.MountStoreWithDB(keyStake, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	err := ms.LoadLatestVersion()
	require.Nil(t, err)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "foochainid"}, isCheckTx, log.NewNopLogger())
	ctx = ctx.WithConsensusParams(&abci.ConsensusParams{Validator: &abci.ValidatorParams{PubKeyTypes: []string{tmtypes.ABCIPubKeyTypeEd25519}}})
	cdc := MakeTestCodec()
	accountKeeper := auth.NewAccountKeeper(
		cdc,                   // amino codec
		keyAcc,                // target store
		auth.ProtoBaseAccount, // prototype
	)

	ck := bank.NewBaseKeeper(accountKeeper)

	pk := params.NewKeeper(cdc, keyParams, tkeyParams)
	keeper := NewKeeper(cdc, keyStake, tkeyStake, ck, pk.Subspace(types.DefaultParamSpace), types.DefaultCodespace, NopMetrics())
	keeper.SetPool(ctx, types.Pool{
		BondedPool: types.InitialBondedPool(),
	})
	keeper.SetParams(ctx, types.DefaultParams())

	// fill all the addresses with some coins, set the loose pool tokens simultaneously
	for _, addr := range Addrs {
		_, _, err := ck.AddCoins(ctx, addr, sdk.Coins{
			{keeper.BondDenom(), initCoins},
		})
		require.Nil(t, err)
		keeper.bankKeeper.IncreaseLoosenToken(ctx, sdk.Coins{
			{keeper.BondDenom(), initCoins},
		})
	}

	return ctx, accountKeeper, keeper
}

func NewPubKey(pk string) (res crypto.PubKey) {
	pkBytes, err := hex.DecodeString(pk)
	if err != nil {
		panic(err)
	}
	//res, err = crypto.PubKeyFromBytes(pkBytes)
	var pkEd ed25519.PubKeyEd25519
	copy(pkEd[:], pkBytes[:])
	return pkEd
}

// for incode address generation
func TestAddr(addr string, bech string) sdk.AccAddress {

	res, err := sdk.AccAddressFromHex(addr)
	if err != nil {
		panic(err)
	}
	bechexpected := res.String()
	if bech != bechexpected {
		panic("Bech encoding doesn't match reference")
	}

	bechres, err := sdk.AccAddressFromBech32(bech)
	if err != nil {
		panic(err)
	}
	if bytes.Compare(bechres, res) != 0 {
		panic("Bech decode and hex decode don't match")
	}

	return res
}

// nolint: unparam
func createTestAddrs(numAddrs int) []sdk.AccAddress {
	var addresses []sdk.AccAddress
	var buffer bytes.Buffer

	// start at 100 so we can make up to 999 test addresses with valid test addresses
	for i := 100; i < (numAddrs + 100); i++ {
		numString := strconv.Itoa(i)
		buffer.WriteString("A58856F0FD53BF058B4909A21AEC019107BA6") //base address string

		buffer.WriteString(numString) //adding on final two digits to make addresses unique
		res, _ := sdk.AccAddressFromHex(buffer.String())
		bech := res.String()
		addresses = append(addresses, TestAddr(buffer.String(), bech))
		buffer.Reset()
	}
	return addresses
}

// nolint: unparam
func createTestPubKeys(numPubKeys int) []crypto.PubKey {
	var publicKeys []crypto.PubKey
	var buffer bytes.Buffer

	//start at 10 to avoid changing 1 to 01, 2 to 02, etc
	for i := 100; i < (numPubKeys + 100); i++ {
		numString := strconv.Itoa(i)
		buffer.WriteString("0B485CFC0EECC619440448436F8FC9DF40566F2369E72400281454CB552AF") //base pubkey string
		buffer.WriteString(numString)                                                       //adding on final two digits to make pubkeys unique
		publicKeys = append(publicKeys, NewPubKey(buffer.String()))
		buffer.Reset()
	}
	return publicKeys
}

//_____________________________________________________________________________________

// does a certain by-power index record exist
func ValidatorByPowerIndexExists(ctx sdk.Context, keeper Keeper, power []byte) bool {
	store := ctx.KVStore(keeper.storeKey)
	return store.Has(power)
}

// update validator for testing
func TestingUpdateValidator(keeper Keeper, ctx sdk.Context, validator types.Validator, apply bool) types.Validator {
	keeper.SetValidator(ctx, validator)
	{ // Remove any existing power key for validator.
		store := ctx.KVStore(keeper.storeKey)
		iterator := sdk.KVStorePrefixIterator(store, ValidatorsByPowerIndexKey)
		deleted := false
		for ; iterator.Valid(); iterator.Next() {
			valAddr := parseValidatorPowerRankKey(iterator.Key())
			if bytes.Equal(valAddr, validator.OperatorAddr) {
				if deleted {
					panic("found duplicate power index key")
				} else {
					deleted = true
				}
				store.Delete(iterator.Key())
			}
		}
	}
	pool := keeper.GetPool(ctx)
	keeper.SetValidatorByPowerIndex(ctx, validator, pool)
	if apply {
		keeper.ApplyAndReturnValidatorSetUpdates(ctx)
		validator, found := keeper.GetValidator(ctx, validator.OperatorAddr)
		if !found {
			panic("validator expected but not found")
		}
		return validator
	}
	cachectx, _ := ctx.CacheContext()
	keeper.ApplyAndReturnValidatorSetUpdates(cachectx)
	validator, found := keeper.GetValidator(cachectx, validator.OperatorAddr)
	if !found {
		panic("validator expected but not found")
	}
	return validator
}

func validatorByPowerIndexExists(k Keeper, ctx sdk.Context, power []byte) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(power)
}

// RandomValidator returns a random validator given access to the keeper and ctx
func RandomValidator(r *rand.Rand, keeper Keeper, ctx sdk.Context) types.Validator {
	vals := keeper.GetAllValidators(ctx)
	i := r.Intn(len(vals))
	return vals[i]
}

// RandomBondedValidator returns a random bonded validator given access to the keeper and ctx
func RandomBondedValidator(r *rand.Rand, keeper Keeper, ctx sdk.Context) types.Validator {
	vals := keeper.GetBondedValidatorsByPower(ctx)
	i := r.Intn(len(vals))
	return vals[i]
}
