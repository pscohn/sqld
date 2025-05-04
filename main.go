package main

// todo
// - test invalid queries and error handling
// 	 - better interface - config + command, end to end test driven by go generate, to see actual interface for errors
//   - ensure operator is valid with left and right, and left/right both same type
//   - remove ToLower calls everwhere with raw string identifier checks
// - should be able to vastly simplify generated simple_select_comparisons.go
// - see if escape hatch is doable
// - allow putting a select query in a fragment
// - set up test suite for running queries directly against a database
// - fmt.fprintf might clean some things up
// - performance pass

// todo - more sql (postgres) support
// - "not" unary: https://www.postgresql.org/docs/current/functions-logical.html
// - comparison predicates: https://www.postgresql.org/docs/current/functions-comparison.html

func main() {
	panic("not implemented")
}
