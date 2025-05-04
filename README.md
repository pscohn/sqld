# sqld

A templating system for building SQL queries in Go.

> [!WARNING]
> sqld is a work in progress and should not be used.

`sqld` relies on code generation to transform sql-like syntax, with additional features for dynamically
generating clauses, into query-building code that builds SQL dynamically based on input.

## Design Goals
- Building queries should look as similar as possible to writing raw SQL,
  but maintaining the benefit of safety that query builders provide.
- Provide more safety than typical query builders with checks during code generation
  that are aware of your schema.
- Provide minimal, fast generated output that does the minimum necessary to build queries.


## Current Status

Currently focused on building full support for Postgres.

- [x] Simple select queries
- [x] Dynamically generated where clauses
- [x] Range loops for dynamically adding clauses
- [x] Fragments for sharing SQL clauses between queries
- [ ] Full support for Postgres SQL

## How to Use

TODO

## Examples

### Define Schema

```sql
CREATE TABLE authors (
  id   BIGSERIAL PRIMARY KEY,
  name text      NOT NULL,
  bio  text
);

```

### Plain SQL Query

No dynamic substitution, but this will validate your field and table names
against your schema:

```sql
query GetAuthorSimpleSelect {
  SELECT id FROM authors
  WHERE id = 5 LIMIT 1
}
```

### Required fields

By default, fields are required and will generate a where clause.

```sql
  query GetAuthorWithVariable(id: string) {
    SELECT id FROM authors
    WHERE id = {id} LIMIT 1
  }
```

Call with this code:

```go
query, args := QueryGetAuthorWithVariable(GetAuthorWithVariableInput{id: "32"})
// query = "SELECT id FROM authors WHERE id = $1 LIMIT 1;"
// args = []interface{}{"32"}
```

### Optional fields

Specify a field as optional with `?`, and that sub-expression can be excluded
from your query if passed as nil.

```sql
  query GetAuthorWithVariable(id: string?) {
    SELECT id FROM authors
    WHERE id = {id}
    LIMIT 1
  }
```

Call with this code:

```go
query, args := QueryGetAuthorWithVariable(GetAuthorWithVariableInput{id: nil})
// query = "SELECT id FROM authors LIMIT 1;"
// args = []interface{}{}
```

This works with more complex nested expression (excuse the contrived example):

```sql
  query GetAuthorWithVariable(id: string?, bio: string?) {
    SELECT id FROM authors
    WHERE id = {id} OR (id > {id} AND (bio = string OR id < {id}))
    LIMIT 1
  }
```

```go
query, args := QueryGetAuthorWithVariable(GetAuthorWithVariableInput{id: "10", bio: nil})
// query = "SELECT id FROM authors WHERE id = $1 OR (id > $2 AND (id < $3)) LIMIT 1;"
// args = []interface{}{"10", "10", "10"}
```

### `if` statements

Any expression can be surrounded with an if statement, which uses the same SQL syntaxes
but only has access to the data in the template itself.

```sql
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
```

```go
query, args := QueryGetAuthorIfStatementMultipleJoined(GetAuthorIfStatementMultipleJoinedInput{bioOptional: ptr("specialValue"), id: nil})
// query = "SELECT id FROM authors WHERE id IS NULL;"
// args = interface{}{}
```

### `foreach` statements

You can iterate over a list input to generate multiple clauses.

```sql
query GetAuthorForLoop(bioLike: string, bioOptional: string?, queries: [string]) {
  SELECT id FROM authors
  WHERE
    bio like {bioLike}
    AND
    (
      {foreach query in queries: AND}
        first_name like {query}
        OR
        last_name like {query}
        OR
        alias like {query}
      {end}
    )
  LIMIT 1
}
```

This will join a clause for each `query` with an AND:

```go
query, args := QueryGetAuthorForLoop(GetAuthorForLoopInput{bioLike: "My bio", queries: []string{"Fred", "Smith"} })
// query = "SELECT id FROM authors WHERE bio LIKE $1 AND 
//            ((first_name LIKE $2 OR last_name LIKE $3 OR alias LIKE $4) 
//            AND 
//            (first_name LIKE $5 OR last_name LIKE $6 OR alias LIKE $7) 
//            LIMIT 1;"
// args = []interface{}{"My bio", "Fred", "Fred", "Fred", "Smith", "Smith", "Smith"}
```

### Fragments

Fragments allow you to share clauses between queries.

```sql
fragment SearchAuthorName(queries: [string]) {
  {foreach query in queries: AND}
    first_name like {query}
    OR
    last_name like {query}
    OR
    alias like {query}
  {end}
}

query GetAuthorForLoop(bioLike: string, bioOptional: string?, queries: [string]) {
  SELECT id FROM authors
  WHERE
    bio like {bioLike}
    AND
    {include SearchAuthorName(queries)}
  LIMIT 1
}
```