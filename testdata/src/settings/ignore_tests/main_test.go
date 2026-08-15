package ignoretests

// collapsibleHelper has a collapsible multi-line signature in a TEST file —
// with ignore-tests: true it must NOT produce a diagnostic (and no fix).
func collapsibleHelper(
	a int,
	b string,
) error {
	return nil
}

// packedHelper mirrors helper's shape; also suppressed by ignore-tests.
func packedHelper(
	param1WithAVeryLongNamesHere string, param2WithAVeryLongNamesHere string, param3WithAVeryLongNamesHere string,
	param4WithAVeryLongNamesHere string, param5WithAVeryLongNamesHere string, param6WithAVeryLongNamesHere string,
) error {
	return nil
}
