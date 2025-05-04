package main

import (
	"fmt"
	"strings"
)

type GetAuthorIfStatementInput struct {
	bioOptional *string
}

func QueryGetAuthorIfStatement(input GetAuthorIfStatementInput) (string, []interface{}) {
	sb := strings.Builder{}
	args := []interface{}{}

	argIndex := 1

	sb.WriteString("SELECT id FROM authors")

	if input.bioOptional == nil {
		lit1 := "bio"
		lit2 := "NULL"
		expr1 := fmt.Sprintf("%s IS %s", lit1, lit2)
		sb.WriteString(fmt.Sprintf(" WHERE %s", expr1))

	} else if *input.bioOptional == "specialValue" {
	} else {
		if input.bioOptional != nil {
			lit3 := "bio"
			lit4 := fmt.Sprintf("$%d", argIndex)
			args = append(args, *input.bioOptional)
			argIndex++
			expr2 := fmt.Sprintf("%s = %s", lit3, lit4)
			sb.WriteString(fmt.Sprintf(" WHERE %s", expr2))

		}

	}

	sb.WriteString(";")

	return sb.String(), args
}
