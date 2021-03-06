// nolint
package tags

import (
	sdk "github.com/NPC-Chain/npcchub/types"
)

var (
	ActionCreateValidator      = []byte("create-validator")
	ActionEditValidator        = []byte("edit-validator")
	ActionDelegate             = []byte("delegate")
	ActionBeginUnbonding       = []byte("begin-unbonding")
	ActionCompleteUnbonding    = []byte("complete-unbonding")
	ActionBeginRedelegation    = []byte("begin-redelegation")
	ActionCompleteRedelegation = []byte("complete-redelegation")

	Action       = sdk.TagAction
	SrcValidator = sdk.TagSrcValidator
	DstValidator = sdk.TagDstValidator
	Delegator    = sdk.TagDelegator
	Moniker      = "moniker"
	Identity     = "identity"
	EndTime      = "end-time"
	Balance      = "balance"
	SharesSrc    = "shares-src"
	SharesDst    = "shares-dst"
)
