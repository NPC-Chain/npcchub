package slashing

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/NPC-Chain/npcchub/types"
)

func TestHookOnValidatorBonded(t *testing.T) {
	ctx, _, _, _, keeper := createTestInput(t, DefaultParamsForTestnet())
	addr := sdk.ConsAddress(addrs[0])
	keeper.onValidatorBonded(ctx, addr, nil)
	period := keeper.getValidatorSlashingPeriodForHeight(ctx, addr, ctx.BlockHeight())
	require.Equal(t, ValidatorSlashingPeriod{addr, ctx.BlockHeight(), 0, sdk.ZeroDec()}, period)
}

func TestHookOnValidatorBeginUnbonding(t *testing.T) {
	ctx, _, _, _, keeper := createTestInput(t, DefaultParamsForTestnet())
	addr := sdk.ConsAddress(addrs[0])
	keeper.onValidatorBonded(ctx, addr, nil)
	keeper.onValidatorBeginUnbonding(ctx, addr, addrs[0])
	period := keeper.getValidatorSlashingPeriodForHeight(ctx, addr, ctx.BlockHeight())
	require.Equal(t, ValidatorSlashingPeriod{addr, ctx.BlockHeight(), ctx.BlockHeight(), sdk.ZeroDec()}, period)
}
