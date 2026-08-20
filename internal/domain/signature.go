package domain

import (
	"go/ast"
	"go/token"
)

// Signature contains metadata about a function signature being analyzed.
type Signature struct {
	Start             token.Pos
	End               token.Pos
	DiagPos           token.Pos
	OneLineText       string
	FuncType          *ast.FuncType
	Receiver          *ast.FieldList
	Name              string
	IsStructField     bool
	IsInterfaceMethod bool

	// SuffixText is the source text that follows End on the signature's last
	// line but is NOT part of the rewritten range — today, a struct tag
	// (` `json:"..."``). It is never emitted or modified; its only role is
	// length budgeting, so a collapsed or packed line keeps the whole
	// construct (signature + suffix) within the line limit.
	SuffixText string
}
