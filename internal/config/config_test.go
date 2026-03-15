package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTokenLimits(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: map[string]int{},
		},
		{
			name:     "single token",
			input:    "abc123:100",
			expected: map[string]int{"abc123": 100},
		},
		{
			name:     "multiple tokens",
			input:    "abc123:100,vip:500,basic:10",
			expected: map[string]int{"abc123": 100, "vip": 500, "basic": 10},
		},
		{
			name:     "whitespace around values",
			input:    " abc123 : 100 , vip : 500 ",
			expected: map[string]int{"abc123": 100, "vip": 500},
		},
		{
			name:     "invalid format skipped",
			input:    "abc123:100,invalid,vip:500",
			expected: map[string]int{"abc123": 100, "vip": 500},
		},
		{
			name:     "non-numeric limit skipped",
			input:    "abc123:notanumber,vip:500",
			expected: map[string]int{"vip": 500},
		},
		{
			name:     "zero limit skipped",
			input:    "abc123:0,vip:500",
			expected: map[string]int{"vip": 500},
		},
		{
			name:     "negative limit skipped",
			input:    "abc123:-10,vip:500",
			expected: map[string]int{"vip": 500},
		},
		{
			name:     "empty token name skipped",
			input:    ":100,vip:500",
			expected: map[string]int{"vip": 500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseTokenLimits(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
