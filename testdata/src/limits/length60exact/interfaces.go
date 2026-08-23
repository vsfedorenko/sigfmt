package length60exact

// Interface method packing budget (limit 60, tab width 8): indent + the
// one-line signature text, with the closing ")" riding free (historical
// tagless budget — see builderLineWriter). DoFit raw = 53 packs fine
// (8 + 53 - 1 = 60, inclusive); DoOver raw = 54 splits — aggressive
// packing never leaves an over-limit one-liner in place.
type IFit interface {
	DoFit(xxxxxxxxxxxxxxxxxxxxxxxxxxx int, second string) // raw = 53 -> fits
	DoOver(xxxxxxxxxxxxxxxxxxxxxxxxxxx int, second string) // want "Signature can be formatted more compactly"
}
