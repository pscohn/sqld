package main

import (
	"fmt"
	"strings"
)

type GetAuthorIfStatementMultipleJoinedInput struct {
	bioOptional *string
	id          *int
}

func QueryGetAuthorIfStatementMultipleJoined(input GetAuthorIfStatementMultipleJoinedInput) (string, []interface{}) {
	sb := strings.Builder{}
	args := []interface{}{}

	argIndex := 1

	sb.WriteString("SELECT id FROM authors")

	groupClause1 := make([]string, 0, 2)

	if input.id == nil {
		lit1 := "id"
		lit2 := "NULL"
		expr1 := fmt.Sprintf("%s IS %s", lit1, lit2)
		groupClause1 = append(groupClause1, expr1)
	}

	if input.bioOptional == nil {
		lit3 := "bio"
		lit4 := "NULL"
		expr2 := fmt.Sprintf("%s IS %s", lit3, lit4)
		groupClause1 = append(groupClause1, expr2)
	} else if *input.bioOptional == "specialValue" {
	} else {
		if input.bioOptional != nil {
			lit5 := "bio"
			lit6 := fmt.Sprintf("$%d", argIndex)
			args = append(args, *input.bioOptional)
			argIndex++
			expr3 := fmt.Sprintf("%s = %s", lit5, lit6)
			groupClause1 = append(groupClause1, expr3)
		}

	}

	groupClause1Result := strings.Join(groupClause1, " AND ")
	if len(groupClause1Result) > 0 {
		sb.WriteString(fmt.Sprintf(" WHERE %s", groupClause1Result))
	}

	sb.WriteString(";")

	return sb.String(), args
}
