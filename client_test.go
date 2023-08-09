package loki

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClipLine(t *testing.T) {
	testCases := []struct {
		name                  string
		testConfig            *TestConfig
		line                  string
		expectedMinLineLength int
		expectedMaxLineLength int
	}{
		{
			name:                  "MaxLineSize",
			testConfig:            &TestConfig{MaxLineSize: 5},
			line:                  "123456",
			expectedMinLineLength: 5,
			expectedMaxLineLength: 5,
		},
		{
			name:                  "RandomLineSize",
			testConfig:            &TestConfig{RandomLineSizeMin: 5, RandomLineSizeMax: 10},
			line:                  "1234567890",
			expectedMinLineLength: 5,
			expectedMaxLineLength: 10,
		},
		{
			name:                  "NoClip",
			testConfig:            &TestConfig{},
			line:                  "1234567890",
			expectedMinLineLength: 10,
			expectedMaxLineLength: 10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualLine := clipLine(tc.testConfig, tc.line)
			assert.LessOrEqual(t, tc.expectedMinLineLength, len(actualLine))
			assert.LessOrEqual(t, len(actualLine), tc.expectedMaxLineLength)
		})
	}
}
