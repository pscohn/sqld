package main

import (
	"fmt"
	"os"
	"path"
)

// todo
// - test invalid queries and error handling
//   - ensure operator is valid with left and right, and left/right both same type
// - see if escape hatch is doable
// - allow putting a select query in a fragment
// - should be able to vastly simplify generated simple_select_comparisons.go
// - set up test suite for running queries directly against a database
// - fmt.fprintf might clean some things up
// - performance pass

// todo - more sql (postgres) support
// - "not" unary: https://www.postgresql.org/docs/current/functions-logical.html
// - comparison predicates: https://www.postgresql.org/docs/current/functions-comparison.html

type Config struct {
	SchemaPath    string
	QueryPath     string
	OutputPath    string
	OutputPackage string
}

func (c *Config) Set(key string, val string) error {
	switch key {
	case "schema_path":
		c.SchemaPath = val
	case "query_path":
		c.QueryPath = val
	case "output_path":
		c.OutputPath = val
	case "output_package":
		c.OutputPackage = val
	default:
		return fmt.Errorf("unknown key: %s", key)
	}
	return nil
}

func (c Config) Validate() error {
	if c.QueryPath == "" {
		return fmt.Errorf("missing query_path")
	}
	if c.SchemaPath == "" {
		return fmt.Errorf("missing schema_path")
	}
	if c.OutputPath == "" {
		return fmt.Errorf("missing output_path")
	}
	if c.OutputPackage == "" {
		return fmt.Errorf("missing output_package")
	}
	return nil
}

func parseConfig(input string) (Config, error) {
	result := Config{}

	scanner := NewScanner(input)
	for scanner.HasNextToken() {

		tokenKey, err := scanner.EatToken()
		if err != nil {
			return result, err
		}

		if tokenKey.Type != Identifier {
			return result, fmt.Errorf("expected identifier")
		}

		token, err := scanner.EatToken()
		if err != nil {
			return result, err
		}

		if token.Type != Equal {
			return result, fmt.Errorf("expected equal sign")
		}

		tokenValue, err := scanner.EatToken()
		if err != nil {
			return result, err
		}

		if tokenValue.Type != String {
			return result, fmt.Errorf("expected string value")
		}

		err = result.Set(tokenKey.Lexeme, tokenValue.Literal.String())
		if err != nil {
			return result, nil
		}
	}

	err := result.Validate()
	if err != nil {
		return result, err
	}

	return result, nil
}

func run() error {
	// look for sqld.conf in current directory

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to read current directory: %w", err)
	}

	filepath := path.Join(dir, "sqld.conf")
	body, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("error reading file at %s: %w", filepath, err)
	}

	config, err := parseConfig(string(body))
	if err != nil {
		return fmt.Errorf("error parsing config: %w", err)
	}

	schema, err := parseSchema(config.SchemaPath)
	if err != nil {
		return fmt.Errorf("error parsing schema: %w", err)
	}

	queryParser, err := parseQueries(config.QueryPath)
	if err != nil {
		return fmt.Errorf("error parsing queries: %w", err)
	}

	checkErrors := CheckQueries(schema, queryParser)
	for _, e := range checkErrors {
		fmt.Fprintf(os.Stderr, "%s", e.Err.Error())
	}

	if len(checkErrors) > 0 {
		return fmt.Errorf("checks failed")
	}

	generated, err := Generate(schema, queryParser, config.OutputPackage)
	if err != nil {
		return fmt.Errorf("error generating: %w", err)
	}

	err = os.WriteFile(config.OutputPath, []byte(generated), 0644)
	if err != nil {
		return fmt.Errorf("error writing output: %w", err)
	}

	return nil
}

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
	}
}
