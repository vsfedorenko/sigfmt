package main

import (
	"context"
	"database/sql"
)

// Too long, needs wrapping. Group [ctx, tx].
func LongQueryFunctionWithManyArguments(
	ctx context.Context,
	tx *sql.Tx,
	query string,
	args ...interface{},
) error { // want "Signature can be formatted more compactly"
	return nil
}

// Too long. Group [ctx].
func LongExecFunctionWithoutTransaction(
	ctx context.Context,
	query string,
	extremelyLongArgumentName string,
) error { // want "Signature can be formatted more compactly"
	return nil
}

// No match for group
func LongNormalFunctionWithArguments(
	a int,
	b int,
	c int,
) { // want "Signature can be formatted more compactly"
}
