package config

const (
	// DefaultMaxLineLen is the default maximum line length allowed before wrapping is suggested.
	DefaultMaxLineLen = 120
	// DefaultTabWidth is the default number of spaces a tab character represents for length calculation.
	DefaultTabWidth = 8
	// DefaultPackStructFields is the default value for packing struct fields.
	DefaultPackStructFields = true
	// DefaultPackInterfaceMethods is the default value for packing interface methods.
	DefaultPackInterfaceMethods = true
)

const (
	keyMaxLineLen           = "max-line-len"
	keyTabWidth             = "tab-width"
	keyPackStructFields     = "pack-struct-fields"
	keyPackInterfaceMethods = "pack-interface-methods"
	keyParamGroups          = "param-groups"
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
	s := defaults()

	m, ok := settings.(map[string]interface{})
	if !ok {
		return s
	}

	s.MaxLineLen = parsePositiveInt(m, keyMaxLineLen, s.MaxLineLen)
	s.TabWidth = parsePositiveInt(m, keyTabWidth, s.TabWidth)
	s.PackStructFields = parseBool(m, keyPackStructFields, s.PackStructFields)
	s.PackInterfaceMethods = parseBool(m, keyPackInterfaceMethods, s.PackInterfaceMethods)
	s.ParamGroups = parseParamGroups(m, keyParamGroups)

	return s
}

// defaults returns a Settings with default values.
func defaults() Settings {
	return Settings{
		MaxLineLen:           DefaultMaxLineLen,
		TabWidth:             DefaultTabWidth,
		PackStructFields:     DefaultPackStructFields,
		PackInterfaceMethods: DefaultPackInterfaceMethods,
	}
}

// parsePositiveInt extracts a positive integer from the map, or returns the default.
func parsePositiveInt(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key].(float64); ok && v > 0 {
		return int(v)
	}
	return defaultValue
}

// parseBool extracts a boolean from the map, or returns the default.
func parseBool(m map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return defaultValue
}

// parseParamGroups extracts parameter groups from the map.
// Each group is a list of type names that should be kept together.
func parseParamGroups(m map[string]interface{}, key string) [][]string {
	groups, ok := m[key].([]interface{})
	if !ok {
		return nil
	}

	var result [][]string
	for _, g := range groups {
		if group := parseStringSlice(g); len(group) > 0 {
			result = append(result, group)
		}
	}
	return result
}

// parseStringSlice converts an interface{} slice to a string slice.
func parseStringSlice(v interface{}) []string {
	group, ok := v.([]interface{})
	if !ok {
		return nil
	}

	var result []string
	for _, item := range group {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}
