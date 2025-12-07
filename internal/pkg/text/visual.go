package text

// VisualLength calculates the visual length of a string, accounting for tab expansion.
// Tabs are expanded to tabWidth spaces.
func VisualLength(s string, tabWidth int) int {
	length := 0
	for _, c := range s {
		if c == '\t' {
			length += tabWidth
		} else {
			length++
		}
	}
	return length
}
