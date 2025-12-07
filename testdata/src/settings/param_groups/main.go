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

// --- Interfaces ---

// Repository interface with param groups
type Repository interface {
	// Group [ctx, tx]
	Create(
		ctx context.Context,
		tx *sql.Tx,
		id int,
		name string,
	) error // want "Signature can be formatted more compactly"

	// Group [ctx]
	Update(
		ctx context.Context,
		data []byte,
	) error // want "Signature can be formatted more compactly"

	// No group match
	Delete(
		id int,
		force bool,
	) error // want "Signature can be formatted more compactly"
}

// --- Struct Fields ---

// Handler struct with function fields
type Handler struct {
	// Group [ctx, tx]
	OnCreate func(
		ctx context.Context,
		tx *sql.Tx,
		data string,
	) error // want "Signature can be formatted more compactly"

	// Group [ctx]
	OnUpdate func(
		ctx context.Context,
		id int,
	) error // want "Signature can be formatted more compactly"

	// No group match
	OnDelete func(
		id int,
		cascade bool,
	) error // want "Signature can be formatted more compactly"
}
