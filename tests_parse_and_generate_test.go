package main

import (
	"errors"
	"go/format"
	"os"
	"testing"
)

func TestGeneration(t *testing.T) {

	schema := `
	CREATE TABLE authors (
		id   BIGSERIAL PRIMARY KEY,
		first_name text      NOT NULL,
		last_name text NOT NULL,
		alias text NOT NULL,
		bio  text
	);
	`

	type testCase struct {
		name             string
		queries          string
		expectErrors     []error
		expectResult     string
		expectResultFile string
	}

	testCases := []testCase{
		{
			name: "simple select",
			queries: `
				query GetAuthorSimpleSelect {
					SELECT id FROM authors
					WHERE id = 5
				}
			`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_simple_select.go",
			// todo: this simpler output would be nice, could even remove string builder.
			// 			expectResult: `import "strings"

			// func QueryGetAuthor() (string, []interface{}) {
			// 	sb := strings.Builder{}
			// 	args := []interface{}{}

			// 	sb.WriteString("select id from authors where id = 5 limit 1")
			// 	return sb.String(), args
			// }
			// `,
		},
		{
			name: "simple select - comparison checks",
			queries: `
				query GetAuthorSimpleSelectComparisons {
					SELECT id FROM authors
					WHERE id = 5 OR id < 5 OR id > 5 OR id != 5 OR id <> 5 OR id <= 5 OR id >= 5
				}
			`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_simple_select_comparisons.go",
		},
		{
			name: "simple select - table and column alias",
			queries: `
				query GetAuthorSimpleSelectAlias {
					SELECT id as my_id FROM authors as my_authors
					WHERE my_authors.id = 5 or id = 5;
				}
			`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_simple_select_alias.go",
		},
		{
			name: "simple select with unknown table",
			queries: `
				query GetAuthorUnknownTable {
					SELECT id FROM authorsWrong
					WHERE id = 5 LIMIT 1
				}
			`,
			expectErrors: []error{ErrUnknownTable},
		},
		{
			name: "simple select with unknown table - aliased",
			queries: `
				query GetAuthorUnknownTableAlias {
					SELECT id FROM authors as "authorsAlias"
					WHERE authors.id = 5 LIMIT 1
				}
			`,
			expectErrors: []error{ErrUnknownTable},
		},
		{
			name: "simple select with wrong select field",
			queries: `
				query GetAuthorWrongSelect {
					SELECT idWrong FROM authors
					WHERE id = 5 LIMIT 1
				}
			`,
			expectErrors: []error{ErrUnknownField},
		},
		{
			name: "simple select with wrong where field",
			queries: `
				query GetAuthorWrongWhere {
					SELECT id FROM authors
					WHERE idWrong = 5 LIMIT 1
				}
			`,
			// todo: perhaps test more specific error, like invalid name, clause it's in, line number
			expectErrors: []error{ErrUnknownField},
		},
		{
			name: "simple select with multiple wrong select field",
			queries: `
				query GetAuthorMultipleWrongSelet {
					SELECT idWrong, idWrong2 FROM authors
					WHERE id = 5 LIMIT 1
				}
			`,
			expectErrors: []error{ErrUnknownField, ErrUnknownField},
		},

		{
			name: "select with variable",
			queries: `
				query GetAuthorWithVariable(id: string!) {
					SELECT id FROM authors
					WHERE id = {id} LIMIT 1
				}
			`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_select_with_variable.go",
		},

		{
			name: "select with optional where",
			queries: `
						query GetAuthorOptionalWhere(id: string?) {
							SELECT id FROM authors
							WHERE id = {id} LIMIT 1
						}
					`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_select_optional_where.go",
		},

		{
			name: "select with multiple optional where - or clause",
			queries: `
						query GetAuthorOptionalWhereOr(id: string?, id2: string?) {
							SELECT id FROM authors
							WHERE id = {id} OR id = {id2} LIMIT 1
						}
					`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_select_optional_where_or.go",
		},

		{
			name: "select with more complex where clause",
			queries: `
						query GetAuthorMoreComplexWhere(id: string?, id2: string?, id3: string?, id4: string? id5: string?) {
							SELECT id FROM authors
							WHERE
								(id = {id} OR (id = {id2} AND (id = {id3} OR id = {id4}) AND id={id5}))
							LIMIT 1
						}
					`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_select_more_complex_where.go",
		},

		{
			name: "select with loop over input",
			queries: `
						query GetAuthorForLoop(bioLike: string, queries: [string], bioOptional: string?) {
							SELECT id FROM authors
							WHERE
								bio like {bioLike}
								AND
								(
									{foreach query in queries: AND}
										bio like {query}
										OR
										first_name like {query}
										OR
										last_name like {query}
										OR
										alias like {query}
									{end}

									AND

									bio like {bioOptional}
								)
							LIMIT 1
						}
					`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_select_for_loop.go",
		},

		{
			name: "select with shared fragment",
			queries: `
				fragment AuthorFragment(bioLike2: string, bioLikeOptional2: string?) {
					bio like {bioLike2}
					or
					bio like {bioLikeOptional2}
				}

				query GetAuthorWithFragment(bioLike: string, bioLikeOptional: string?) {
					SELECT id FROM authors
					WHERE
						id = 1
						AND
						{include AuthorFragment(bioLike, bioLikeOptional)}
					LIMIT 1
				}
			`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_select_fragment.go",
		},

		{
			name: "select with shared fragment - errors with unknown fragment",
			queries: `
				fragment AuthorFragment(bioLike: string, bioLikeOptional: string?) {
					bio like {bioLike}
					or
					bio like {bioLikeOptional}
				}

				query GetAuthorWithFragment(bioLike: string, bioLikeOptional: string?) {
					SELECT id FROM authors
					WHERE
						id = 1
						AND
						{include AuthorFragmentMispelled(bioLike)}

					LIMIT 1
				}
			`,
			expectErrors:     []error{ErrUnknownFragment},
			expectResultFile: "",
		},

		{
			name: "select with shared fragment - errors when parameters don't match",
			queries: `
				fragment AuthorFragment(bioLike: string, bioLikeOptional: string?) {
					bio like {bioLike}
					or
					bio like {bioLikeOptional}
				}

				query GetAuthorWithFragment(bioLike: string, bioLikeOptional: string?) {
					SELECT id FROM authors
					WHERE
						id = 1
						AND
						{include AuthorFragment(bioLike)}

					LIMIT 1
				}
			`,
			expectErrors:     []error{ErrFragmentParamMismatch},
			expectResultFile: "",
		},

		{
			name: "select with shared fragment - errors when parameters have different types",
			queries: `
				fragment AuthorFragment(bioLike: int, bioLikeOptional: string?) {
					bio like {bioLike}
					or
					bio like {bioLikeOptional}
				}

				query GetAuthorWithFragment(bioLike: string, bioLikeOptional: string?) {
					SELECT id FROM authors
					WHERE
						id = 1
						AND
						{include AuthorFragment(bioLike, bioLikeOptional)}

					LIMIT 1
				}
			`,
			expectErrors:     []error{ErrFragmentParamMismatch},
			expectResultFile: "",
		},

		{
			name: "select with if statement - equality",
			queries: `
						query GetAuthorIfStatement(bioOptional: string?) {
							SELECT id FROM authors
							WHERE
								{if bioOptional IS NULL}
									bio IS NULL
								{else if bioOptional = "specialValue"}
								{else}
									bio = {bioOptional}
								{end}
						}
					`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_select_if_statement_equality.go",
		},

		{
			name: "select with if statement - multiple joined",
			queries: `
						query GetAuthorIfStatementMultipleJoined(bioOptional: string?, id: int?) {
							SELECT id FROM authors
							WHERE
								{if id is NULL}
									id IS NULL
								{end}

								AND

								{if bioOptional IS NULL}
									bio IS NULL
								{else if bioOptional = "specialValue"}
								{else}
									bio = {bioOptional}
								{end}
						}
					`,
			expectErrors:     nil,
			expectResultFile: "tests_sample_select_if_statement_multiple_joined.go",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			schemaParser := NewSchemaParser(schema)
			schemaParser.Parse()

			queryParser := NewQueryParser(test.queries)
			queryParser.Parse()

			checkErrors := CheckQueries(schemaParser.Result, queryParser.Result)
			if len(test.expectErrors) != len(checkErrors) {
				t.Log(test.expectErrors, checkErrors)
				t.Fatalf("expected %d errors, got %d", len(test.expectErrors), len(checkErrors))
			} else {
				for i := range checkErrors {
					if !errors.Is(checkErrors[i].Err, test.expectErrors[i]) {
						t.Fatalf("expected error: %s, got: %s", test.expectErrors[i], checkErrors[i].Err)
					}
				}
			}

			if len(checkErrors) == 0 {
				generated, err := Generate(schemaParser.Result, queryParser.Result, "main")
				if err != nil {
					// allow continuing in case it's an error while formatting
					t.Errorf("got error: %s", err)
				}

				expectedResult := test.expectResult

				if test.expectResultFile != "" {
					file, err := os.ReadFile(test.expectResultFile)
					if err != nil {
						t.Fatalf("error reading file: %s", err)
					}
					expectedResult = string(file)
				}

				expectedResultFormatted, err := format.Source([]byte(expectedResult))
				if err != nil {
					t.Fatalf("error formatting: %s", err)
				}
				expectedResult = string(expectedResultFormatted)

				if generated != expectedResult {
					_ = os.WriteFile("testfailure/expected.go", []byte(expectedResult), 0644)
					_ = os.WriteFile("testfailure/got.go", []byte(generated), 0644)
					t.Errorf("generated output mismatch")
					t.Log("expected:\n\n", expectedResult)
					t.Log("got:\n\n", generated)
				}
			}
		})
	}
}

func ptr[T any](in T) *T {
	return &in
}

func assertQuery(t *testing.T, expectQuery string, expectArgs []interface{}, gotQuery string, gotArgs []interface{}) {
	if expectQuery != gotQuery {
		t.Fatalf("\nexpected qry: %s\ngot:          %s", expectQuery, gotQuery)
	}

	if len(expectArgs) != len(gotArgs) {
		t.Fatalf("expected %d args, got %d", len(expectArgs), len(gotArgs))
	} else {
		for i := range expectArgs {
			if expectArgs[i] != gotArgs[i] {
				t.Fatalf("\nexpected arg: %s\ngot:          %s", expectArgs[i], gotArgs[i])
			}
		}
	}
}

// test the generated output with various inputs to
// ensure they produce expected queries
func TestGeneratedSelects(t *testing.T) {
	t.Run("simple select", func(t *testing.T) {
		query, args := QueryGetAuthorSimpleSelect()
		assertQuery(t,
			"SELECT id FROM authors WHERE id = 5;",
			[]interface{}{},
			query,
			args,
		)
	})

	t.Run("simple select - table and column alias", func(t *testing.T) {
		query, args := QueryGetAuthorSimpleSelectAlias()
		assertQuery(t,
			"SELECT id my_id FROM authors my_authors WHERE my_authors.id = 5 OR id = 5;",
			[]interface{}{},
			query,
			args,
		)
	})

	t.Run("select with variable", func(t *testing.T) {
		query, args := QueryGetAuthorWithVariable(GetAuthorWithVariableInput{id: "32"})
		assertQuery(t,
			"SELECT id FROM authors WHERE id = $1 LIMIT 1;",
			[]interface{}{"32"},
			query,
			args,
		)
	})
	t.Run("select with variable - no input", func(t *testing.T) {
		// todo: consider if this should error instead
		query, args := QueryGetAuthorWithVariable(GetAuthorWithVariableInput{})
		assertQuery(t,
			"SELECT id FROM authors WHERE id = $1 LIMIT 1;",
			[]interface{}{""},
			query,
			args,
		)
	})

	t.Run("select optional where - nil parameter", func(t *testing.T) {
		query, args := QueryGetAuthorOptionalWhere(GetAuthorOptionalWhereInput{})
		assertQuery(t,
			"SELECT id FROM authors LIMIT 1;",
			[]interface{}{},
			query,
			args,
		)
	})

	t.Run("select optional where - non-nil parameter", func(t *testing.T) {
		query, args := QueryGetAuthorOptionalWhere(GetAuthorOptionalWhereInput{id: ptr("23")})
		assertQuery(t,
			"SELECT id FROM authors WHERE id = $1 LIMIT 1;",
			[]interface{}{"23"},
			query,
			args,
		)
	})

	t.Run("select optional where or - nil parameters", func(t *testing.T) {
		query, args := QueryGetAuthorOptionalWhereOr(GetAuthorOptionalWhereOrInput{})
		assertQuery(t,
			"SELECT id FROM authors LIMIT 1;",
			[]interface{}{},
			query,
			args,
		)
	})
	t.Run("select optional where or - non-nil", func(t *testing.T) {
		query, args := QueryGetAuthorOptionalWhereOr(GetAuthorOptionalWhereOrInput{id: ptr("8"), id2: ptr("9")})
		assertQuery(t,
			"SELECT id FROM authors WHERE id = $1 OR id = $2 LIMIT 1;",
			[]interface{}{"8", "9"},
			query,
			args,
		)
	})
	t.Run("select optional where or - id2 is nil", func(t *testing.T) {
		query, args := QueryGetAuthorOptionalWhereOr(GetAuthorOptionalWhereOrInput{id: ptr("8"), id2: nil})
		assertQuery(t,
			"SELECT id FROM authors WHERE id = $1 LIMIT 1;",
			[]interface{}{"8"},
			query,
			args,
		)
	})
	t.Run("select optional where or - id1 is nil", func(t *testing.T) {
		query, args := QueryGetAuthorOptionalWhereOr(GetAuthorOptionalWhereOrInput{id: nil, id2: ptr("9")})
		assertQuery(t,
			"SELECT id FROM authors WHERE id = $1 LIMIT 1;",
			[]interface{}{"9"},
			query,
			args,
		)
	})

	t.Run("select with loop", func(t *testing.T) {
		query, args := QueryGetAuthorForLoop(GetAuthorForLoopInput{bioLike: "bio", queries: []string{"foo", "bar"}, bioOptional: ptr("opt")})
		assertQuery(t,
			"SELECT id FROM authors WHERE bio LIKE $1 AND (((((bio LIKE $2 OR first_name LIKE $3) OR last_name LIKE $4) OR alias LIKE $5) AND (((bio LIKE $6 OR first_name LIKE $7) OR last_name LIKE $8) OR alias LIKE $9)) AND bio LIKE $10) LIMIT 1;",
			[]interface{}{"bio", "foo", "foo", "foo", "foo", "bar", "bar", "bar", "bar", "opt"},
			query,
			args,
		)
	})
	t.Run("select with loop - nil arg", func(t *testing.T) {
		query, args := QueryGetAuthorForLoop(GetAuthorForLoopInput{bioLike: "bio", queries: []string{"foo", "bar"}, bioOptional: nil})
		assertQuery(t,
			"SELECT id FROM authors WHERE bio LIKE $1 AND (((((bio LIKE $2 OR first_name LIKE $3) OR last_name LIKE $4) OR alias LIKE $5) AND (((bio LIKE $6 OR first_name LIKE $7) OR last_name LIKE $8) OR alias LIKE $9))) LIMIT 1;",
			[]interface{}{"bio", "foo", "foo", "foo", "foo", "bar", "bar", "bar", "bar"},
			query,
			args,
		)
	})
	t.Run("select with loop - nil arg and one item in list", func(t *testing.T) {
		query, args := QueryGetAuthorForLoop(GetAuthorForLoopInput{bioLike: "bio", queries: []string{"foo"}, bioOptional: nil})
		assertQuery(t,
			"SELECT id FROM authors WHERE bio LIKE $1 AND (((((bio LIKE $2 OR first_name LIKE $3) OR last_name LIKE $4) OR alias LIKE $5))) LIMIT 1;",
			[]interface{}{"bio", "foo", "foo", "foo", "foo"},
			query,
			args,
		)
	})
	t.Run("select with loop - nil arg and empty list", func(t *testing.T) {
		query, args := QueryGetAuthorForLoop(GetAuthorForLoopInput{bioLike: "bio", queries: []string{}, bioOptional: nil})
		assertQuery(t,
			"SELECT id FROM authors WHERE bio LIKE $1 LIMIT 1;",
			[]interface{}{"bio"},
			query,
			args,
		)
	})

	t.Run("select with fragment - optional provided", func(t *testing.T) {
		query, args := QueryGetAuthorWithFragment(GetAuthorWithFragmentInput{bioLike: "bio", bioLikeOptional: ptr("bio2")})
		assertQuery(t,
			"SELECT id FROM authors WHERE id = 1 AND (bio LIKE $1 OR bio LIKE $2) LIMIT 1;",
			[]interface{}{"bio", "bio2"},
			query,
			args,
		)
	})
	t.Run("select with fragment - optional not provided", func(t *testing.T) {
		query, args := QueryGetAuthorWithFragment(GetAuthorWithFragmentInput{bioLike: "bio", bioLikeOptional: nil})
		assertQuery(t,
			"SELECT id FROM authors WHERE id = 1 AND (bio LIKE $1) LIMIT 1;",
			[]interface{}{"bio"},
			query,
			args,
		)
	})

	// query GetAuthorIfStatement(bioOptional: string?) {
	// 	SELECT id FROM authors
	// 		WHERE
	// 			{if bioOptional == nil}
	// 			bio IS NULL
	// 			{else if bioOptional == "specialValue"}
	// 			{else}
	// 			bio = {bioLike}
	// 			{end}
	// }
	t.Run("select with if statement equality - is null check", func(t *testing.T) {
		query, args := QueryGetAuthorIfStatement(GetAuthorIfStatementInput{bioOptional: nil})
		assertQuery(t,
			"SELECT id FROM authors WHERE bio IS NULL;",
			[]interface{}{},
			query,
			args,
		)
	})
	t.Run("select with if statement equality - empty statement", func(t *testing.T) {
		query, args := QueryGetAuthorIfStatement(GetAuthorIfStatementInput{bioOptional: ptr("specialValue")})
		assertQuery(t,
			"SELECT id FROM authors;",
			[]interface{}{},
			query,
			args,
		)
	})
	t.Run("select with if statement equality - any other value", func(t *testing.T) {
		query, args := QueryGetAuthorIfStatement(GetAuthorIfStatementInput{bioOptional: ptr("bio")})
		assertQuery(t,
			"SELECT id FROM authors WHERE bio = $1;",
			[]interface{}{"bio"},
			query,
			args,
		)
	})

	// query GetAuthorIfStatementMultipleJoined(bioOptional: string?, id: int?) {
	// 	SELECT id FROM authors
	// 	WHERE
	// 		{if id is NULL}
	// 			id IS NULL
	// 		{end}

	// 		AND

	// 		{if bioOptional IS NULL}
	// 			bio IS NULL
	// 		{else if bioOptional = "specialValue"}
	// 		{else}
	// 			bio = {bioOptional}
	// 		{end}
	// }
	t.Run("select with if statement equality - is null check", func(t *testing.T) {
		query, args := QueryGetAuthorIfStatementMultipleJoined(GetAuthorIfStatementMultipleJoinedInput{bioOptional: nil, id: nil})
		assertQuery(t,
			"SELECT id FROM authors WHERE id IS NULL AND bio IS NULL;",
			[]interface{}{},
			query,
			args,
		)
	})
	t.Run("select with if statement equality - empty statement", func(t *testing.T) {
		query, args := QueryGetAuthorIfStatementMultipleJoined(GetAuthorIfStatementMultipleJoinedInput{bioOptional: ptr("specialValue"), id: nil})
		assertQuery(t,
			"SELECT id FROM authors WHERE id IS NULL;",
			[]interface{}{},
			query,
			args,
		)
	})
	t.Run("select with if statement equality - multiple empty statements", func(t *testing.T) {
		query, args := QueryGetAuthorIfStatementMultipleJoined(GetAuthorIfStatementMultipleJoinedInput{bioOptional: ptr("specialValue"), id: ptr(3)})
		assertQuery(t,
			"SELECT id FROM authors;",
			[]interface{}{},
			query,
			args,
		)
	})
	t.Run("select with if statement equality - empty ID check", func(t *testing.T) {
		query, args := QueryGetAuthorIfStatementMultipleJoined(GetAuthorIfStatementMultipleJoinedInput{bioOptional: ptr("bio"), id: ptr(3)})
		assertQuery(t,
			"SELECT id FROM authors WHERE bio = $1;",
			[]interface{}{"bio"},
			query,
			args,
		)
	})

}

func TestGeneratedSelectMoreComplexWhere(t *testing.T) {
	/*
		query GetAuthor(id: string?, id2: string?, id3: string?, id4: string? id5: string?) {
			SELECT id FROM authors
			WHERE
				id = {id}
				OR
				(
					id = {id2} AND (id = {id3} OR id = {id4})
					AND
					id={id5}
				)
			LIMIT 1
	*/

	type testCase struct {
		name        string
		input       GetAuthorMoreComplexWhereInput
		expectQuery string
		expectArgs  []interface{}
	}

	testCases := []testCase{
		{
			name:        "no params",
			input:       GetAuthorMoreComplexWhereInput{},
			expectQuery: "SELECT id FROM authors LIMIT 1;",
			expectArgs:  []interface{}{},
		},
		{
			name:        "id1 only",
			input:       GetAuthorMoreComplexWhereInput{id: ptr("4")},
			expectQuery: "SELECT id FROM authors WHERE id = $1 LIMIT 1;",
			expectArgs:  []interface{}{"4"},
		},
		// todo: remove extraneous parentheses, or make it more consistent
		{
			name:        "id5 only",
			input:       GetAuthorMoreComplexWhereInput{id5: ptr("4")},
			expectQuery: "SELECT id FROM authors WHERE (id = $1) LIMIT 1;",
			expectArgs:  []interface{}{"4"},
		},
		{
			name:        "id1 and id5",
			input:       GetAuthorMoreComplexWhereInput{id: ptr("2"), id5: ptr("4")},
			expectQuery: "SELECT id FROM authors WHERE id = $1 OR (id = $2) LIMIT 1;",
			expectArgs:  []interface{}{"2", "4"},
		},
		{
			name:        "all ids",
			input:       GetAuthorMoreComplexWhereInput{id: ptr("1"), id2: ptr("2"), id3: ptr("3"), id4: ptr("4"), id5: ptr("5")},
			expectQuery: "SELECT id FROM authors WHERE id = $1 OR ((id = $2 AND (id = $3 OR id = $4)) AND id = $5) LIMIT 1;",
			expectArgs:  []interface{}{"1", "2", "3", "4", "5"},
		},
	}

	for i, test := range testCases {
		// shortcut to isolate tests
		if i < 0 {
			continue
		}
		t.Run(test.name, func(t *testing.T) {
			query, args := QueryGetAuthorMoreComplexWhere(test.input)
			assertQuery(t, test.expectQuery, test.expectArgs, query, args)
		})
	}
}
