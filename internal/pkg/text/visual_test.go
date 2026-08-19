package text

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVisualLength(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		tabWidth int
		want     int
	}{
		{"Empty", "", 8, 0},
		{"NoTabs", "hello", 8, 5},
		{"OneTab", "\thello", 8, 13},
		{"MultipleTabs", "\t\thello", 8, 21},
		{"Spaces", "    hello", 8, 9},
		{"Mixed", "\t hello", 8, 14},
		{"TabWidth4", "\thello", 4, 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VisualLength(tt.input, tt.tabWidth)
			assert.Equal(t, tt.want, got, "VisualLength(%q, %d)", tt.input, tt.tabWidth)
		})
	}
}
