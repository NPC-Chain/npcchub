//nolint
package bank

import (
	sdk "github.com/NPC-Chain/npcchub/types"
)

// Bank errors reserve 100 ~ 199.
const (
	DefaultCodespace sdk.CodespaceType = "bank"

	CodeInvalidInput   sdk.CodeType = 101
	CodeInvalidOutput  sdk.CodeType = 102
	CodeBurnEmptyCoins sdk.CodeType = 103
)

// NOTE: Don't stringer this, we'll put better messages in later.
func codeToDefaultMsg(code sdk.CodeType) string {
	switch code {
	case CodeInvalidInput:
		return "invalid input coins"
	case CodeInvalidOutput:
		return "invalid output coins"
	case CodeBurnEmptyCoins:
		return "burn empty coins"
	default:
		return sdk.CodeToDefaultMsg(code)
	}
}

//----------------------------------------
// Error constructors

func ErrInvalidInput(codespace sdk.CodespaceType, msg string) sdk.Error {
	return newError(codespace, CodeInvalidInput, msg)
}

func ErrNoInputs(codespace sdk.CodespaceType) sdk.Error {
	return newError(codespace, CodeInvalidInput, "")
}

func ErrInvalidOutput(codespace sdk.CodespaceType, msg string) sdk.Error {
	return newError(codespace, CodeInvalidOutput, msg)
}

func ErrNoOutputs(codespace sdk.CodespaceType) sdk.Error {
	return newError(codespace, CodeInvalidOutput, "")
}

func ErrBurnEmptyCoins(codespace sdk.CodespaceType) sdk.Error {
	return newError(codespace, CodeBurnEmptyCoins, "")
}

//----------------------------------------

func msgOrDefaultMsg(msg string, code sdk.CodeType) string {
	if msg != "" {
		return msg
	}
	return codeToDefaultMsg(code)
}

func newError(codespace sdk.CodespaceType, code sdk.CodeType, msg string) sdk.Error {
	msg = msgOrDefaultMsg(msg, code)
	return sdk.NewError(codespace, code, msg)
}
