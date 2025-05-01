package main

import (
	"fmt"
	"strings"
)

type GetAuthorOptionalWhereOrInput struct {
	id  *string
	id2 *string
}

func QueryGetAuthorOptionalWhereOr(input GetAuthorOptionalWhereOrInput) (string, []interface{}) {
	sb := strings.Builder{}
	args := []interface{}{}

	argIndex := 1

	sb.WriteString("SELECT id FROM authors")

	groupClause1 := make([]string, 0, 2)

	if input.id != nil {
		lit1 := "id"
		lit2 := fmt.Sprintf("$%d", argIndex)
		args = append(args, *input.id)
		argIndex++
		expr1 := fmt.Sprintf("%s = %s", lit1, lit2)
		groupClause1 = append(groupClause1, expr1)
	}

	if input.id2 != nil {
		lit3 := "id"
		lit4 := fmt.Sprintf("$%d", argIndex)
		args = append(args, *input.id2)
		argIndex++
		expr2 := fmt.Sprintf("%s = %s", lit3, lit4)
		groupClause1 = append(groupClause1, expr2)
	}

	groupClause1Result := strings.Join(groupClause1, " OR ")
	if len(groupClause1Result) > 0 {
		sb.WriteString(fmt.Sprintf(" WHERE %s", groupClause1Result))
	}

	sb.WriteString(" LIMIT 1")
	sb.WriteString(";")

	return sb.String(), args
}
