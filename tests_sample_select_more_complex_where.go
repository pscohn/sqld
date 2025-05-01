package main

import (
	"fmt"
	"strings"
)

type GetAuthorMoreComplexWhereInput struct {
	id  *string
	id2 *string
	id3 *string
	id4 *string
	id5 *string
}

func QueryGetAuthorMoreComplexWhere(input GetAuthorMoreComplexWhereInput) (string, []interface{}) {
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

	groupClause2 := make([]string, 0, 2)

	groupClause3 := make([]string, 0, 2)

	if input.id2 != nil {
		lit3 := "id"
		lit4 := fmt.Sprintf("$%d", argIndex)
		args = append(args, *input.id2)
		argIndex++
		expr2 := fmt.Sprintf("%s = %s", lit3, lit4)
		groupClause3 = append(groupClause3, expr2)
	}

	groupClause4 := make([]string, 0, 2)

	if input.id3 != nil {
		lit5 := "id"
		lit6 := fmt.Sprintf("$%d", argIndex)
		args = append(args, *input.id3)
		argIndex++
		expr3 := fmt.Sprintf("%s = %s", lit5, lit6)
		groupClause4 = append(groupClause4, expr3)
	}

	if input.id4 != nil {
		lit7 := "id"
		lit8 := fmt.Sprintf("$%d", argIndex)
		args = append(args, *input.id4)
		argIndex++
		expr4 := fmt.Sprintf("%s = %s", lit7, lit8)
		groupClause4 = append(groupClause4, expr4)
	}

	groupClause4Result := strings.Join(groupClause4, " OR ")
	if len(groupClause4Result) > 0 {
		groupClause3 = append(groupClause3, fmt.Sprintf("(%s)", groupClause4Result))
	}

	groupClause3Result := strings.Join(groupClause3, " AND ")
	if len(groupClause3Result) > 0 {
		groupClause2 = append(groupClause2, fmt.Sprintf("(%s)", groupClause3Result))
	}

	if input.id5 != nil {
		lit9 := "id"
		lit10 := fmt.Sprintf("$%d", argIndex)
		args = append(args, *input.id5)
		argIndex++
		expr5 := fmt.Sprintf("%s = %s", lit9, lit10)
		groupClause2 = append(groupClause2, expr5)
	}

	groupClause2Result := strings.Join(groupClause2, " AND ")
	if len(groupClause2Result) > 0 {
		groupClause1 = append(groupClause1, fmt.Sprintf("(%s)", groupClause2Result))
	}

	groupClause1Result := strings.Join(groupClause1, " OR ")
	if len(groupClause1Result) > 0 {
		sb.WriteString(fmt.Sprintf(" WHERE %s", groupClause1Result))
	}

	sb.WriteString(" LIMIT 1")
	sb.WriteString(";")

	return sb.String(), args
}
