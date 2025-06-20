package main

import (
	"fmt"
	"go/format"
	"strings"
)

// todo: reconsider which fields/methods are exposed
type Generator struct {
	PackageName string
	Schema      Schema

	// Used for giving unique names to output of expressions
	GroupIndex   int
	ExprIndex    int
	LiteralIndex int

	// If a where clause is added dynamically based on conditionals or presence of values,
	// use this to add the WHERE keyword in top-level expressions.
	// TODO: this isn't the best way to do this, since expressions should ideally not care about what
	// context it's used in. We could instead write end result to variable in all cases, and
	// add the WHERE keyword if expression value is not empty.
	GenPossiblyOptionalWhereClause bool
}

// handles generating ops within template expressions (Go syntax).
// eventually we abstracted pattern for generators targeting different languages
func OpTypeToGoString(opType OpType) string {
	var op string
	switch opType {
	case OpTypeAnd:
		op = "&&"
	case OpTypeOr:
		op = "||"
	case OpTypeEquals:
		op = "=="
	case OpTypeNotEquals:
		op = "!="
	case OpTypeLess:
		op = "<"
	case OpTypeGreater:
		op = ">"
	case OpTypeLessOrEqual:
		op = "<="
	case OpTypeGreaterOrEqual:
		op = ">="
	case OpTypeLike:
		panic("not supported")
	case OpTypeNotLike:
		panic("not supported")
	case OpTypeIs:
		op = "=="
	case OpTypeIsNot:
		op = "!="
	default:
		panic("unhandled op")
	}
	return op
}

func (g *Generator) writeTemplateExpressionLiteral(sb *strings.Builder, params []Param, exp Expression, isPointerComparison bool) {
	switch exp.LiteralType {
	case LiteralTypeNull:
		sb.WriteString("nil")
	case LiteralTypeString:
		sb.WriteString(fmt.Sprintf("\"%s\"", exp.LiteralString))
	case LiteralTypeNumber:
		sb.WriteString(fmt.Sprintf("%d", exp.LiteralNumber))
	case LiteralTypeFieldName:
		panic("invalid literal type for template expression")
	case LiteralTypeVariable:
		maybePointer := ""
		if !exp.IsClauseRequired && !isPointerComparison {
			maybePointer = "*"
		}

		sb.WriteString(fmt.Sprintf("%s%s", maybePointer, exp.LiteralVariableName))
	default:
		panic("unhandled literal type")
	}
}

func (g *Generator) writeTemplateExpressionBinary(sb *strings.Builder, params []Param, exp Expression) {
	op := OpTypeToGoString(exp.Op)

	isPointerComparison := false
	if exp.Left.LiteralType == LiteralTypeNull || exp.Right.LiteralType == LiteralTypeNull {
		isPointerComparison = true
	}

	sb.WriteString("(")
	g.writeTemplateExpressionLiteral(sb, params, *exp.Left, isPointerComparison)
	sb.WriteString(fmt.Sprintf(" %s ", op))
	g.writeTemplateExpressionLiteral(sb, params, *exp.Right, isPointerComparison)
	sb.WriteString(")")
}

func (g *Generator) startGroup(sb *strings.Builder) {
	g.GroupIndex++
	sb.WriteString(fmt.Sprintf("\tgroupClause%d := make([]string, 0, 2)\n\n", g.GroupIndex))
}

func (g *Generator) endGroup(sb *strings.Builder, groupIndex int, op string, addToGroupClauseNum *int) {
	sb.WriteString(fmt.Sprintf("\tgroupClause%dResult := strings.Join(groupClause%d, \" %s \")\n", groupIndex, groupIndex, op))
	sb.WriteString(fmt.Sprintf("\tif len(groupClause%dResult) > 0 {\n", groupIndex))
	if addToGroupClauseNum != nil {
		sb.WriteString(fmt.Sprintf("\t\tgroupClause%d = append(groupClause%d, fmt.Sprintf(\"(%%s)\", groupClause%dResult))\n", *addToGroupClauseNum, *addToGroupClauseNum, groupIndex))
	} else {
		// this is the top level expression, so add the base where clause
		possibleWhere := ""
		if g.GenPossiblyOptionalWhereClause {
			possibleWhere = " WHERE "
		}
		sb.WriteString(fmt.Sprintf("sb.WriteString(fmt.Sprintf(\"%s%%s\", groupClause%dResult))", possibleWhere, groupIndex))
	}
}

func (g *Generator) writeLiteral(sb *strings.Builder, params []Param, exp Expression) {
	switch exp.LiteralType {
	case LiteralTypeNull:
		g.LiteralIndex++
		sb.WriteString(fmt.Sprintf("\tlit%d := \"NULL\"\n", g.LiteralIndex))
	case LiteralTypeString:
		g.LiteralIndex++
		sb.WriteString(fmt.Sprintf("\tlit%d := \"'%s'\"\n", g.LiteralIndex, exp.LiteralString))
	case LiteralTypeNumber:
		g.LiteralIndex++
		sb.WriteString(fmt.Sprintf("\tlit%d := \"%d\"\n", g.LiteralIndex, exp.LiteralNumber))
	case LiteralTypeFieldName:
		g.LiteralIndex++
		sb.WriteString(fmt.Sprintf("\tlit%d := \"", g.LiteralIndex))

		if exp.LiteralField.TableName != "" {
			sb.WriteString(exp.LiteralField.TableName)
			sb.WriteString(".")
		}
		if exp.LiteralField.All {
			sb.WriteString("*")
		} else {
			sb.WriteString(exp.LiteralField.Name)
		}
		if exp.LiteralField.Alias != "" {
			sb.WriteString(" ")
			sb.WriteString(exp.LiteralField.Alias)
		}

		sb.WriteString("\"\n")
	case LiteralTypeVariable:
		maybePointer := ""
		if !exp.IsClauseRequired {
			maybePointer = "*"
		}
		g.LiteralIndex++

		sb.WriteString(fmt.Sprintf("\tlit%d := fmt.Sprintf(\"$%%d\", argIndex)\n", g.LiteralIndex))
		sb.WriteString(fmt.Sprintf("\targs = append(args, %s", maybePointer))
		sb.WriteString(exp.LiteralVariableName)
		sb.WriteString(")\n\targIndex++\n")
	default:
		panic("unhandled literal type")
	}
}

func (g *Generator) writeBinary(sb *strings.Builder, params []Param, exp Expression, addToGroupClauseNum *int) {
	// todo: consider using a lexeme we should already have validated
	op := exp.Op.String()

	if exp.Left.Type == ExpressionTypeLiteral && exp.Right.Type == ExpressionTypeLiteral {
		// should not start a group, but clause may not be required.
		// need to know whether one of children has a variable so we can add fmt.Sprintf

		usesVar := false
		if exp.Left.LiteralType == LiteralTypeVariable && !exp.Left.IsClauseRequired {
			usesVar = true
			sb.WriteString(fmt.Sprintf("\tif %s != nil {\n", exp.Left.LiteralVariableName))
		} else if exp.Right.LiteralType == LiteralTypeVariable && !exp.Right.IsClauseRequired {
			usesVar = true
			sb.WriteString(fmt.Sprintf("\tif %s != nil {\n", exp.Right.LiteralVariableName))
		}

		g.writeLiteral(sb, params, *exp.Left)
		g.writeLiteral(sb, params, *exp.Right)

		// literal needs to be able to write arg either in first position or second
		// so the literals each generate separate string variable for now, and compose them here
		g.ExprIndex++
		exprName := fmt.Sprintf("expr%d", g.ExprIndex)
		sb.WriteString(fmt.Sprintf("\t%s := fmt.Sprintf(\"%%s %s %%s\", lit%d, lit%d)\n", exprName, op, g.LiteralIndex-1, g.LiteralIndex))

		// todo: some duplication here with endGroup
		if addToGroupClauseNum != nil {
			sb.WriteString(fmt.Sprintf("\tgroupClause%d = append(groupClause%d, %s)\n", *addToGroupClauseNum, *addToGroupClauseNum, exprName))
		} else {
			possibleWhere := ""
			if g.GenPossiblyOptionalWhereClause {
				possibleWhere = " WHERE "
			}
			sb.WriteString(fmt.Sprintf("sb.WriteString(fmt.Sprintf(\"%s%%s\", %s))\n\n", possibleWhere, exprName))
		}

		if usesVar {
			sb.WriteString("\t}\n\n")
		}

	} else {
		// left and right expressions need to be added to dynamically
		g.startGroup(sb)

		groupIndex := g.GroupIndex

		g.writeExpression(sb, params, *exp.Left, &groupIndex)
		g.writeExpression(sb, params, *exp.Right, &groupIndex)

		g.endGroup(sb, groupIndex, op, addToGroupClauseNum)

		sb.WriteString("\t}\n\n")
	}
}

func (g *Generator) writeForLoop(sb *strings.Builder, params []Param, exp Expression, addToGroupClauseNum *int) {
	g.GroupIndex++
	sb.WriteString(fmt.Sprintf("\tgroupClause%d := make([]string, 0, len(input.%s))\n\n", g.GroupIndex, exp.ForLoopVarName))

	// todo: ForLoopIteratorName should have a unique suffix? because if the loop variable is passed to
	// a child fragment, that child fragment should be able to both reference the loop variable
	// and define a new loop variable with the same name.
	sb.WriteString(fmt.Sprintf("\tfor _, %s := range input.%s {\n", exp.ForLoopIteratorName, exp.ForLoopVarName))

	groupIndex := g.GroupIndex
	op := "AND"
	if exp.ForLoopJoinByOr {
		op = "OR"
	}

	g.writeExpression(sb, params, *exp.Left, &groupIndex)

	sb.WriteString("\t}\n\n")

	g.endGroup(sb, groupIndex, op, addToGroupClauseNum)

	sb.WriteString("\t}\n\n")
}

func (g *Generator) writeIf(sb *strings.Builder, params []Param, exp Expression, addToGroupClauseNum *int) {
	for i, elseif := range exp.ElseIfs {
		exprStart := "} else if "
		if i == 0 {
			exprStart = "if "
		}
		sb.WriteString(exprStart)

		// write literals to string
		// could be a deeply nested expression
		// kind of want to call writeBinary but output to a static string
		g.writeTemplateExpressionBinary(sb, params, *elseif.IfExpr)

		sb.WriteString(" {\n")

		if elseif.BodyExpr != nil {
			g.writeExpression(sb, params, *elseif.BodyExpr, addToGroupClauseNum)
		}
	}

	if exp.ElseBody != nil {
		sb.WriteString("} else {\n")
		g.writeExpression(sb, params, *exp.ElseBody, addToGroupClauseNum)
	}

	sb.WriteString("\t}\n\n")
}

// todo: probably will want some object to encapuslate context
// of params, current table, etc
func (g *Generator) writeExpression(sb *strings.Builder, params []Param, exp Expression, addToGroupClauseNum *int) {
	switch exp.Type {
	case ExpressionTypeLiteral:
		g.writeLiteral(sb, params, exp)
	case ExpressionTypeBinary:
		g.writeBinary(sb, params, exp, addToGroupClauseNum)
	case ExpressionTypeForLoop:
		g.writeForLoop(sb, params, exp, addToGroupClauseNum)
	case ExpressionTypeIf:
		g.writeIf(sb, params, exp, addToGroupClauseNum)
	case ExpressionTypeFragment:
		panic("expected fragment to be expanded into expression")
	default:
		panic("unexpected expression type")
	}
}

func (g *Generator) generateQuery(query Query) ([]byte, error) {

	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("package %s\n\n", g.PackageName))
	sb.WriteString("import (\n\"strings\"\n\"fmt\"\n)\n\n")

	if len(query.Params) > 0 {
		sb.WriteString("type ")
		sb.WriteString(query.Name)
		sb.WriteString("Input struct {\n")
		for _, p := range query.Params {
			sb.WriteString("\t")
			sb.WriteString(p.Name)
			sb.WriteString(" ")
			if p.IsList {
				sb.WriteString("[]")
			} else if !p.Required {
				sb.WriteString("*")
			}
			sb.WriteString(p.Type.String())
			sb.WriteString("\n")
		}
		sb.WriteString("}\n\n")
	}

	sb.WriteString("func Query")

	sb.WriteString(query.Name)

	// todo: preallocate args based on number of params to query?
	if len(query.Params) > 0 {
		sb.WriteString("(input ")
		sb.WriteString(query.Name)
		sb.WriteString("Input)")
	} else {
		sb.WriteString("()")
	}
	sb.WriteString(" (string, []interface{}) {\n")
	sb.WriteString("\tsb := strings.Builder{}\n")
	sb.WriteString("\targs := []interface{}{}\n\n")

	if len(query.Params) > 0 {
		sb.WriteString("\targIndex := 1\n\n")
	}

	writeFields := func(fields []Field) {
		for i, f := range fields {
			if i > 0 {
				sb.WriteString(", ")
			}
			if f.TableName != "" {
				sb.WriteString(fmt.Sprintf("%s.", f.TableName))
			}
			sb.WriteString(f.Name)
			if f.Alias != "" {
				sb.WriteString(fmt.Sprintf(" %s", f.Alias))
			}
		}
	}

	switch query.StatementType {
	case StatementTypeSelect:
		sb.WriteString("\tsb.WriteString(\"SELECT ")
		writeFields(query.Select.Fields)

		sb.WriteString(" FROM ")
		sb.WriteString(query.Select.From)
		if query.Select.FromAlias != "" {
			sb.WriteString(fmt.Sprintf(" %s", query.Select.FromAlias))
		}
		sb.WriteString("\")\n\n")

		for _, j := range query.Select.Joins {
			joinType := j.JoinType
			sb.WriteString(fmt.Sprintf("\tsb.WriteString(\" %s %s %s ON \")\n", joinType, j.Table, j.TableAlias))

			g.writeExpression(&sb, query.Params, j.On, nil)
		}

		if query.Select.Where.Type != ExpressionTypeNone {
			g.GenPossiblyOptionalWhereClause = true
			g.writeExpression(&sb, query.Params, query.Select.Where, nil)
			g.GenPossiblyOptionalWhereClause = false
		}

		if len(query.Select.OrderByFields) > 0 {
			sb.WriteString(" ORDER BY ")
			writeFields(query.Select.OrderByFields)
		}

		if query.Select.Limit != nil {
			sb.WriteString("\tsb.WriteString(\" LIMIT ")
			sb.WriteString(fmt.Sprintf("%d", *query.Select.Limit))
			sb.WriteString("\")\n")
		}

		sb.WriteString("sb.WriteString(\";\")\n\n")

	default:
		panic("only selects are supported")
	}

	sb.WriteString("\treturn sb.String(), args\n")
	sb.WriteString("}\n")

	return []byte(sb.String()), nil
}

func Generate(schema Schema, queries Queries, outputPackage string) (string, error) {
	// todo: preallocate
	result := []byte{}
	var err error

	for _, q := range queries.Queries {
		if q.IsFragment {
			continue
		}
		g := Generator{}
		g.PackageName = outputPackage
		g.Schema = schema
		result, err = g.generateQuery(q)
		if err != nil {
			panic("")
		}
	}

	formattedResult, err := format.Source(result)
	if err != nil {
		// return unformatted result for tests
		return string(result), err
	}

	return string(formattedResult), nil
}
