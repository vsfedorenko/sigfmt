package length60exact

// Struct func-field packing budget (limit 60, tab width 8): indent + the
// one-line signature text, closing ")" rides free — same contract as
// interface methods. FnFit raw = 53 stays; FnOver raw = 54 splits into
// the packed two-line shape.
type SFit struct {
	FnFit func(xxxxxxxxxxxxxxxxxxxxxx int, second string) // raw = 53 -> fits
	FnOver func(xxxxxxxxxxxxxxxxxxxxxx int, second string) // want "Signature can be formatted more compactly"
}
