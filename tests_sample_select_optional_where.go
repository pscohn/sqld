package main

import (
	"fmt"
	"strings"
)

type GetAuthorOptionalWhereInput struct {
	id *string
}

func QueryGetAuthorOptionalWhere(input GetAuthorOptionalWhereInput) (string, []interface{}) {
	sb := strings.Builder{}
	args := []interface{}{}

	argIndex := 1

	sb.WriteString("SELECT id FROM authors")

	if input.id != nil {
		lit1 := "id"
		lit2 := fmt.Sprintf("$%d", argIndex)
		args = append(args, *input.id)
		argIndex++
		expr1 := fmt.Sprintf("%s = %s", lit1, lit2)
		sb.WriteString(fmt.Sprintf(" WHERE %s", expr1))

	}

	sb.WriteString(" LIMIT 1")
	sb.WriteString(";")

	return sb.String(), args
}
