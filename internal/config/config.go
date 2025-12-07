package config

const (
	// DefaultMaxLineLen is the default maximum line length allowed before wrapping is suggested.
	DefaultMaxLineLen = 120
	// DefaultTabWidth is the default number of spaces a tab character represents for length calculation.
	DefaultTabWidth = 8
)

// Settings contains configuration parameters for the linter.
type Settings struct {
	// MaxLineLen is the maximum allowed length of a line (including indentation).
	MaxLineLen int
	// TabWidth is the visual width of a tab character used for length calculations.
	TabWidth int
	// PackStructFields enables aggressive packing of function types within structs.
	// If true, multiple parameters will be placed on the same line if they fit.
	PackStructFields bool
	// PackInterfaceMethods enables aggressive packing of method signatures within interfaces.
	// If true, multiple parameters will be placed on the same line if they fit.
	PackInterfaceMethods bool
	// ParamGroups defines strict grouping rules based on type names.
	// If a sequence of parameters matches a group, they are kept together
	// and followed by a line break.
	ParamGroups [][]string
}

// New creates a Settings struct from a generic map.
func New(settings any) Settings {
	s := Settings{
		MaxLineLen:           DefaultMaxLineLen,
		TabWidth:             DefaultTabWidth,
		PackStructFields:     true,
		PackInterfaceMethods: true,
	}

	if m, ok := settings.(map[string]interface{}); ok {
		if v, ok := m["max-line-len"].(float64); ok && v > 0 {
			s.MaxLineLen = int(v)
		}
		if v, ok := m["tab-width"].(float64); ok && v > 0 {
			s.TabWidth = int(v)
		}
		if v, ok := m["pack-struct-fields"].(bool); ok {
			s.PackStructFields = v
		}
		if v, ok := m["pack-interface-methods"].(bool); ok {
			s.PackInterfaceMethods = v
		}
		if groups, ok := m["param-groups"].([]interface{}); ok {
			for _, g := range groups {
				if group, ok := g.([]interface{}); ok {
					var strGroup []string
					for _, item := range group {
						if str, ok := item.(string); ok {
							strGroup = append(strGroup, str)
						}
					}
					if len(strGroup) > 0 {
						s.ParamGroups = append(s.ParamGroups, strGroup)
					}
				}
			}
		}
	}
	return s
}
