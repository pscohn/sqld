package main

import (
	"os"
	"testing"
)

func TestParseSchema(t *testing.T) {

	type testCase struct {
		name             string
		schemaFile       string
		expectTableCount int
		expectErrors     []error
	}

	testCases := []testCase{
		{
			name:             "simple select",
			schemaFile:       "tests_sample_schema.sql",
			expectTableCount: 3,
			expectErrors:     nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			file, err := os.ReadFile(test.schemaFile)
			if err != nil {
				t.Fatalf("error reading file: %s", err)
			}

			schemaParser := NewSchemaParser(string(file))
			schemaParser.Parse()

			if len(schemaParser.Result.Tables) != test.expectTableCount {
				t.Errorf("expected %d tables, got %d", test.expectTableCount, len(schemaParser.Result.Tables))
			}
		})
	}
}
