package main

import (
	"fmt"
	"strings"
)

type GetAuthorWithFragmentInput struct {
	bioLike         string
	bioLikeOptional *string
}

func QueryGetAuthorWithFragment(input GetAuthorWithFragmentInput) (string, []interface{}) {
	sb := strings.Builder{}
	args := []interface{}{}

	argIndex := 1

	sb.WriteString("SELECT id FROM authors")

	groupClause1 := make([]string, 0, 2)

	lit1 := "id"
	lit2 := "1"
	expr1 := fmt.Sprintf("%s = %s", lit1, lit2)
	groupClause1 = append(groupClause1, expr1)
	groupClause2 := make([]string, 0, 2)

	lit3 := "bio"
	lit4 := fmt.Sprintf("$%d", argIndex)
	args = append(args, input.bioLike)
	argIndex++
	expr2 := fmt.Sprintf("%s LIKE %s", lit3, lit4)
	groupClause2 = append(groupClause2, expr2)
	if input.bioLikeOptional != nil {
		lit5 := "bio"
		lit6 := fmt.Sprintf("$%d", argIndex)
		args = append(args, *input.bioLikeOptional)
		argIndex++
		expr3 := fmt.Sprintf("%s LIKE %s", lit5, lit6)
		groupClause2 = append(groupClause2, expr3)
	}

	groupClause2Result := strings.Join(groupClause2, " OR ")
	if len(groupClause2Result) > 0 {
		groupClause1 = append(groupClause1, fmt.Sprintf("(%s)", groupClause2Result))
	}

	groupClause1Result := strings.Join(groupClause1, " AND ")
	if len(groupClause1Result) > 0 {
		sb.WriteString(fmt.Sprintf(" WHERE %s", groupClause1Result))
	}

	sb.WriteString(" LIMIT 1")
	sb.WriteString(";")

	return sb.String(), args
}
