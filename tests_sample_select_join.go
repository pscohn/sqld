package main

import (
	"fmt"
	"strings"
)

func QueryGetAuthorJoin() (string, []interface{}) {
	sb := strings.Builder{}
	args := []interface{}{}

	sb.WriteString("SELECT a1.id FROM authors a1")

	sb.WriteString(" LEFT JOIN authors a2 ON ")
	lit1 := "a1.id"
	lit2 := "a2.id"
	expr1 := fmt.Sprintf("%s = %s", lit1, lit2)
	sb.WriteString(fmt.Sprintf("%s", expr1))

	sb.WriteString(";")

	return sb.String(), args
}
