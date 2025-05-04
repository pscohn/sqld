package main

import (
	"fmt"
	"strings"
)

func QueryGetAuthorSimpleSelectComparisons() (string, []interface{}) {
	sb := strings.Builder{}
	args := []interface{}{}

	sb.WriteString("SELECT id FROM authors")

	groupClause1 := make([]string, 0, 2)

	groupClause2 := make([]string, 0, 2)

	groupClause3 := make([]string, 0, 2)

	groupClause4 := make([]string, 0, 2)

	groupClause5 := make([]string, 0, 2)

	groupClause6 := make([]string, 0, 2)

	lit1 := "id"
	lit2 := "5"
	expr1 := fmt.Sprintf("%s = %s", lit1, lit2)
	groupClause6 = append(groupClause6, expr1)
	lit3 := "id"
	lit4 := "5"
	expr2 := fmt.Sprintf("%s < %s", lit3, lit4)
	groupClause6 = append(groupClause6, expr2)
	groupClause6Result := strings.Join(groupClause6, " OR ")
	if len(groupClause6Result) > 0 {
		groupClause5 = append(groupClause5, fmt.Sprintf("(%s)", groupClause6Result))
	}

	lit5 := "id"
	lit6 := "5"
	expr3 := fmt.Sprintf("%s > %s", lit5, lit6)
	groupClause5 = append(groupClause5, expr3)
	groupClause5Result := strings.Join(groupClause5, " OR ")
	if len(groupClause5Result) > 0 {
		groupClause4 = append(groupClause4, fmt.Sprintf("(%s)", groupClause5Result))
	}

	lit7 := "id"
	lit8 := "5"
	expr4 := fmt.Sprintf("%s != %s", lit7, lit8)
	groupClause4 = append(groupClause4, expr4)
	groupClause4Result := strings.Join(groupClause4, " OR ")
	if len(groupClause4Result) > 0 {
		groupClause3 = append(groupClause3, fmt.Sprintf("(%s)", groupClause4Result))
	}

	lit9 := "id"
	lit10 := "5"
	expr5 := fmt.Sprintf("%s != %s", lit9, lit10)
	groupClause3 = append(groupClause3, expr5)
	groupClause3Result := strings.Join(groupClause3, " OR ")
	if len(groupClause3Result) > 0 {
		groupClause2 = append(groupClause2, fmt.Sprintf("(%s)", groupClause3Result))
	}

	lit11 := "id"
	lit12 := "5"
	expr6 := fmt.Sprintf("%s <= %s", lit11, lit12)
	groupClause2 = append(groupClause2, expr6)
	groupClause2Result := strings.Join(groupClause2, " OR ")
	if len(groupClause2Result) > 0 {
		groupClause1 = append(groupClause1, fmt.Sprintf("(%s)", groupClause2Result))
	}

	lit13 := "id"
	lit14 := "5"
	expr7 := fmt.Sprintf("%s >= %s", lit13, lit14)
	groupClause1 = append(groupClause1, expr7)
	groupClause1Result := strings.Join(groupClause1, " OR ")
	if len(groupClause1Result) > 0 {
		sb.WriteString(fmt.Sprintf(" WHERE %s", groupClause1Result))
	}

	sb.WriteString(";")

	return sb.String(), args
}
