package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromoteLoneDiff(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want []string
	}{
		{
			name: "lone diff is promoted",
			argv: []string{"sigfmt", "-diff", "./..."},
			want: []string{"sigfmt", "-fix", "-diff", "./..."},
		},
		{
			name: "double dash diff is promoted",
			argv: []string{"sigfmt", "--diff", "./..."},
			want: []string{"sigfmt", "-fix", "--diff", "./..."},
		},
		{
			name: "diff with explicit true is promoted",
			argv: []string{"sigfmt", "-diff=true", "./..."},
			want: []string{"sigfmt", "-fix", "-diff=true", "./..."},
		},
		{
			name: "fix and diff stays untouched",
			argv: []string{"sigfmt", "-fix", "-diff", "./..."},
			want: []string{"sigfmt", "-fix", "-diff", "./..."},
		},
		{
			name: "fix after diff stays untouched",
			argv: []string{"sigfmt", "-diff", "-fix", "./..."},
			want: []string{"sigfmt", "-diff", "-fix", "./..."},
		},
		{
			name: "diff=false is not promoted",
			argv: []string{"sigfmt", "-diff=false", "./..."},
			want: []string{"sigfmt", "-diff=false", "./..."},
		},
		{
			name: "fix=false does not count as fix",
			argv: []string{"sigfmt", "-fix=false", "-diff", "./..."},
			want: []string{"sigfmt", "-fix=false", "-fix", "-diff", "./..."},
		},
		{
			name: "value flag before diff is skipped",
			argv: []string{"sigfmt", "-max-line-len", "120", "-diff", "./..."},
			want: []string{"sigfmt", "-max-line-len", "120", "-fix", "-diff", "./..."},
		},
		{
			name: "inline value flag before diff",
			argv: []string{"sigfmt", "-max-line-len=80", "--diff", "./..."},
			want: []string{"sigfmt", "-max-line-len=80", "-fix", "--diff", "./..."},
		},
		{
			name: "diff hidden behind positional arg is not promoted",
			argv: []string{"sigfmt", "./pkg", "-diff"},
			want: []string{"sigfmt", "./pkg", "-diff"},
		},
		{
			name: "diff hidden behind -- separator is not promoted",
			argv: []string{"sigfmt", "--", "-diff"},
			want: []string{"sigfmt", "--", "-diff"},
		},
		{
			name: "no flags at all",
			argv: []string{"sigfmt", "./..."},
			want: []string{"sigfmt", "./..."},
		},
		{
			name: "argv with program only",
			argv: []string{"sigfmt"},
			want: []string{"sigfmt"},
		},
		{
			name: "empty argv",
			argv: []string{},
			want: []string{},
		},
		{
			name: "value flag at end without value",
			argv: []string{"sigfmt", "-max-line-len"},
			want: []string{"sigfmt", "-max-line-len"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := promoteLoneDiff(tt.argv)
			assert.Equal(t, tt.want, got)
		})
	}
}
