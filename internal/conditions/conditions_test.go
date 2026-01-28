package conditions

import (
	"testing"
)

func TestFormatConditions(t *testing.T) {
	tests := []struct {
		name     string
		conds    []Condition
		expected string
	}{
		{
			name:     "empty conditions returns happy",
			conds:    []Condition{},
			expected: "happy",
		},
		{
			name:     "single condition",
			conds:    []Condition{CondLonely},
			expected: "lonely",
		},
		{
			name:     "two conditions",
			conds:    []Condition{CondLonely, CondHungry},
			expected: "lonely, hungry",
		},
		{
			name:     "three conditions",
			conds:    []Condition{CondLonely, CondHungry, CondTired},
			expected: "lonely, hungry, tired",
		},
		{
			name:     "all conditions with message",
			conds:    []Condition{CondHasMessage, CondInfirm, CondLonely, CondHungry, CondTired, CondSad, CondHappy},
			expected: "infirm, lonely, hungry, tired, sad, happy and has a message",
		},
		{
			name:     "duplicate conditions",
			conds:    []Condition{CondLonely, CondLonely, CondHungry},
			expected: "lonely, lonely, hungry",
		},
		{
			name:     "single happy condition",
			conds:    []Condition{CondHappy},
			expected: "happy",
		},
		{
			name:     "stone and infirm - infirm ignored",
			conds:    []Condition{CondStone, CondInfirm},
			expected: "stone",
		},
		{
			name:     "conditions with hyphens",
			conds:    []Condition{CondHasMessage},
			expected: "has-message",
		},
		{
			name:     "nil slice",
			conds:    nil,
			expected: "happy",
		},
		{
			name:     "stone alone",
			conds:    []Condition{CondStone},
			expected: "stone",
		},
		{
			name:     "stone with other conditions ignores others",
			conds:    []Condition{CondStone, CondLonely, CondHungry},
			expected: "stone",
		},
		{
			name:     "stone with has-message",
			conds:    []Condition{CondStone, CondHasMessage},
			expected: "stone and has a message",
		},
		{
			name:     "stone with has-message and other conditions",
			conds:    []Condition{CondStone, CondHasMessage, CondLonely, CondHungry},
			expected: "stone and has a message",
		},
		{
			name:     "has-message without stone",
			conds:    []Condition{CondHasMessage, CondLonely},
			expected: "lonely and has a message",
		},
		{
			name:     "stone appears after other conditions",
			conds:    []Condition{CondLonely, CondHungry, CondStone},
			expected: "stone",
		},
		{
			name:     "stone appears after has-message",
			conds:    []Condition{CondHasMessage, CondLonely, CondStone},
			expected: "stone and has a message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatConditions(tt.conds)
			if result != tt.expected {
				t.Errorf("FormatConditions(%v) = %q, want %q", tt.conds, result, tt.expected)
			}
		})
	}
}
