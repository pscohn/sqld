package main

import (
	"fmt"
	"strings"
)

func QueryGetAuthorSimpleSelect() (string, []interface{}) {
	sb := strings.Builder{}
	args := []interface{}{}

	sb.WriteString("SELECT id FROM authors")

	lit1 := "id"
	lit2 := "5"
	expr1 := fmt.Sprintf("%s = %s", lit1, lit2)
	sb.WriteString(fmt.Sprintf(" WHERE %s", expr1))

	sb.WriteString(";")

	return sb.String(), args
}
