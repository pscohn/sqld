package main

import (
	"fmt"
	"strings"
)

func QueryGetAuthorSimpleSelectAlias() (string, []interface{}) {
	sb := strings.Builder{}
	args := []interface{}{}

	sb.WriteString("SELECT id my_id FROM authors my_authors")

	groupClause1 := make([]string, 0, 2)

	lit1 := "my_authors.id"
	lit2 := "5"
	expr1 := fmt.Sprintf("%s = %s", lit1, lit2)
	groupClause1 = append(groupClause1, expr1)
	lit3 := "id"
	lit4 := "5"
	expr2 := fmt.Sprintf("%s = %s", lit3, lit4)
	groupClause1 = append(groupClause1, expr2)
	groupClause1Result := strings.Join(groupClause1, " OR ")
	if len(groupClause1Result) > 0 {
		sb.WriteString(fmt.Sprintf(" WHERE %s", groupClause1Result))
	}

	sb.WriteString(";")

	return sb.String(), args
}
