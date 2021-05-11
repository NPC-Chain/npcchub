package subspace

import (
	"fmt"
	"testing"

	"github.com/NPC-Chain/npcchub/codec"
	sdk "github.com/NPC-Chain/npcchub/types"
	"github.com/stretchr/testify/require"
)

type testparams struct {
	i int64
	b bool
}

func (tp *testparams) String() string {
	return fmt.Sprintf(`Test Params:
  i:         %d
  b:         %v`,
		tp.i, tp.b)
}

func (tp *testparams) KeyValuePairs() KeyValuePairs {
	return KeyValuePairs{
		{[]byte("i"), &tp.i},
		{[]byte("b"), &tp.b},
	}
}

// Implements params.ParamStruct
func (p *testparams) GetParamSpace() string {
	return "test"
}

func (p *testparams) Validate(key string, value string) (interface{}, sdk.Error) {

	return nil, nil

}

func (p *testparams) StringFromBytes(cdc *codec.Codec, key string, bytes []byte) (string, error) {
	return "", nil
}

func TestTypeTable(t *testing.T) {
	table := NewTypeTable()

	require.Panics(t, func() { table.RegisterType([]byte(""), nil) })
	require.Panics(t, func() { table.RegisterType([]byte("!@#$%"), nil) })
	require.Panics(t, func() { table.RegisterType([]byte("hello,"), nil) })
	require.Panics(t, func() { table.RegisterType([]byte("hello"), nil) })

	require.NotPanics(t, func() { table.RegisterType([]byte("hello"), bool(false)) })
	require.NotPanics(t, func() { table.RegisterType([]byte("world"), int64(0)) })
	require.Panics(t, func() { table.RegisterType([]byte("hello"), bool(false)) })

	require.NotPanics(t, func() { table.RegisterParamSet(&testparams{}) })
	require.Panics(t, func() { table.RegisterParamSet(&testparams{}) })
}
