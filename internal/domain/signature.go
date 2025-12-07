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
}
