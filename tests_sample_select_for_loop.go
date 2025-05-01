package main

import (
	"fmt"
	"strings"
)

type GetAuthorForLoopInput struct {
	bioLike     string
	queries     []string
	bioOptional *string
}

func QueryGetAuthorForLoop(input GetAuthorForLoopInput) (string, []interface{}) {
	sb := strings.Builder{}
	args := []interface{}{}

	argIndex := 1

	sb.WriteString("SELECT id FROM authors")

	groupClause1 := make([]string, 0, 2)

	lit1 := "bio"
	lit2 := fmt.Sprintf("$%d", argIndex)
	args = append(args, input.bioLike)
	argIndex++
	expr1 := fmt.Sprintf("%s LIKE %s", lit1, lit2)
	groupClause1 = append(groupClause1, expr1)
	groupClause2 := make([]string, 0, 2)

	groupClause3 := make([]string, 0, len(input.queries))

	for _, local1_query := range input.queries {
		groupClause4 := make([]string, 0, 2)

		groupClause5 := make([]string, 0, 2)

		groupClause6 := make([]string, 0, 2)

		lit3 := "bio"
		lit4 := fmt.Sprintf("$%d", argIndex)
		args = append(args, local1_query)
		argIndex++
		expr2 := fmt.Sprintf("%s LIKE %s", lit3, lit4)
		groupClause6 = append(groupClause6, expr2)
		lit5 := "first_name"
		lit6 := fmt.Sprintf("$%d", argIndex)
		args = append(args, local1_query)
		argIndex++
		expr3 := fmt.Sprintf("%s LIKE %s", lit5, lit6)
		groupClause6 = append(groupClause6, expr3)
		groupClause6Result := strings.Join(groupClause6, " OR ")
		if len(groupClause6Result) > 0 {
			groupClause5 = append(groupClause5, fmt.Sprintf("(%s)", groupClause6Result))
		}

		lit7 := "last_name"
		lit8 := fmt.Sprintf("$%d", argIndex)
		args = append(args, local1_query)
		argIndex++
		expr4 := fmt.Sprintf("%s LIKE %s", lit7, lit8)
		groupClause5 = append(groupClause5, expr4)
		groupClause5Result := strings.Join(groupClause5, " OR ")
		if len(groupClause5Result) > 0 {
			groupClause4 = append(groupClause4, fmt.Sprintf("(%s)", groupClause5Result))
		}

		lit9 := "alias"
		lit10 := fmt.Sprintf("$%d", argIndex)
		args = append(args, local1_query)
		argIndex++
		expr5 := fmt.Sprintf("%s LIKE %s", lit9, lit10)
		groupClause4 = append(groupClause4, expr5)
		groupClause4Result := strings.Join(groupClause4, " OR ")
		if len(groupClause4Result) > 0 {
			groupClause3 = append(groupClause3, fmt.Sprintf("(%s)", groupClause4Result))
		}

	}

	groupClause3Result := strings.Join(groupClause3, " AND ")
	if len(groupClause3Result) > 0 {
		groupClause2 = append(groupClause2, fmt.Sprintf("(%s)", groupClause3Result))
	}

	if input.bioOptional != nil {
		lit11 := "bio"
		lit12 := fmt.Sprintf("$%d", argIndex)
		args = append(args, *input.bioOptional)
		argIndex++
		expr6 := fmt.Sprintf("%s LIKE %s", lit11, lit12)
		groupClause2 = append(groupClause2, expr6)
	}

	groupClause2Result := strings.Join(groupClause2, " AND ")
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
