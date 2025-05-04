package main

import (
	"errors"
	"fmt"
)

var (
	ErrUnknownTable          = errors.New("unknown table")
	ErrUnknownField          = errors.New("unknown field")
	ErrUnknownParam          = errors.New("unknown param")
	ErrUnknownFragment       = errors.New("unknown fragment")
	ErrFragmentParamMismatch = errors.New("mismatched fragment params")
)

type CheckError struct {
	Err error
}

// todo: consider moving this into struct
var LocalIndex int

// available scope at any given level of the tree:
// - top-level query or fragment params
// - current "local" variables eg the variable defined in a for loop
// - mapping between any fragment params to their global name
type Scope struct {
	Fragments   []Query
	QueryParams []Param
	// todo: likely needs a rework to handle shadowing correctly, this would be "first one ones"
	Locals []Param

	QueryParamToGlobalName map[string]string
}

func checkTable(schema Schema, table string) (Table, CheckError) {
	// todo: consider not looping
	for _, tableDef := range schema.Tables {
		if tableDef.Name == table {
			return tableDef, CheckError{}
		}
	}
	return Table{}, CheckError{
		Err: fmt.Errorf("%w: table %s not found in schema", ErrUnknownTable, table),
	}
}

func checkField(table Table, field string) (TableField, CheckError) {
	// todo: consider not looping
	for _, fieldDef := range table.Fields {
		if fieldDef.Name == field {
			return fieldDef, CheckError{}
		}
	}
	return TableField{}, CheckError{
		Err: fmt.Errorf("%w: field %s not found in table %s", ErrUnknownField, field, table.Name),
	}
}

func checkFragment(scope Scope, fragmentName string) (Query, CheckError) {
	// todo: consider not looping
	for _, fragment := range scope.Fragments {
		if fragment.Name == fragmentName {
			return fragment, CheckError{}
		}
	}
	return Query{}, CheckError{
		Err: fmt.Errorf("%w: fragment %s not found", ErrUnknownFragment, fragmentName),
	}
}

func checkParam(scope Scope, field string) (Param, CheckError) {
	// todo: consider not looping
	for _, param := range scope.Locals {
		if param.Name == field {
			return param, CheckError{}
		}
	}
	for _, param := range scope.QueryParams {
		if param.Name == field {
			return param, CheckError{}
		}
	}
	return Param{}, CheckError{
		Err: fmt.Errorf("%w: param %s not found", ErrUnknownParam, field),
	}
}

// This mutates the expression's IsClauseRequired field.
func checkExpr(table Table, scope Scope, expr *Expression) (*Expression, []CheckError) {
	var errors []CheckError

	switch expr.Type {
	case ExpressionTypeForLoop:
		newScope := scope
		param, e := checkParam(newScope, expr.ForLoopVarName)
		if e.Err != nil {
			errors = append(errors, e)
		}
		if !param.IsList {
			errors = append(errors, CheckError{Err: fmt.Errorf("expected range variable %s to be a list", param.Name)})
		}

		// add for loop iterator to local scope for subexpressions
		LocalIndex++
		iteratorName := fmt.Sprintf("local%d_%s", LocalIndex, expr.ForLoopIteratorName)

		newScope.Locals = make([]Param, len(scope.Locals)+1)
		// todo: see about avoiding copies
		copy(newScope.Locals, scope.Locals)
		newScope.Locals[len(newScope.Locals)-1] = Param{
			Name:       expr.ForLoopIteratorName,
			Type:       param.Type,
			Required:   true,
			GlobalName: iteratorName,
		}

		expr.ForLoopIteratorName = iteratorName

		exprLeft, exprLeftErrors := checkExpr(table, newScope, expr.Left)
		expr.Left = exprLeft

		errors = append(errors, exprLeftErrors...)

	case ExpressionTypeIf:
		for _, elseif := range expr.ElseIfs {
			ifExpr, ifExprErrs := checkExpr(table, scope, elseif.IfExpr)
			errors = append(errors, ifExprErrs...)
			elseif.IfExpr = ifExpr

			if elseif.BodyExpr != nil {
				bodyExpr, bodyExprErrs := checkExpr(table, scope, elseif.BodyExpr)
				errors = append(errors, bodyExprErrs...)
				elseif.BodyExpr = bodyExpr
			}
		}

		if expr.ElseBody != nil {
			bodyExpr, bodyExprErrs := checkExpr(table, scope, expr.ElseBody)
			errors = append(errors, bodyExprErrs...)
			expr.ElseBody = bodyExpr
		}

	case ExpressionTypeFragment:
		fragment, e := checkFragment(scope, expr.FragmentName)
		if e.Err != nil {
			errors = append(errors, e)
			return expr, errors
		}

		if len(fragment.Params) != len(expr.FragmentArgs) {
			errors = append(errors, CheckError{Err: fmt.Errorf("%w: number of params do not match", ErrFragmentParamMismatch)})
			return expr, errors
		}
		for i := range fragment.Params {
			expressionArg, e := checkParam(scope, expr.FragmentArgs[i])
			if e.Err != nil {
				errors = append(errors, e)
			}

			if fragment.Params[i].Type != expressionArg.Type {
				errors = append(errors, CheckError{Err: fmt.Errorf("%w: param type mismatch", ErrFragmentParamMismatch)})
				return expr, errors
			}
		}

		// at this point, fragment is validated.
		// need to create a mapping between fragment params and global names,
		// then traverse fragment (getting back an expression), and return this expression to the parent.
		// parent needs to replace the expression it passed in with any expression returned,
		// since it could be a fragment call we want to substitute

		newScope := Scope{
			Fragments:              scope.Fragments,
			QueryParamToGlobalName: make(map[string]string),
			Locals:                 []Param{},
			QueryParams:            fragment.Params,
		}

		for i, param := range fragment.Params {
			// todo: this is already called above
			inputArg, _ := checkParam(scope, expr.FragmentArgs[i])
			if inputArg.GlobalName == "" {
				panic("expected global name to be set")
			}
			// todo: is this needed, or should we populate the global name of the QueryParams instead
			newScope.QueryParamToGlobalName[param.Name] = inputArg.GlobalName
		}

		fragmentExpr, exprErrors := checkExpr(table, newScope, &fragment.FragmentExpression)
		errors = append(errors, exprErrors...)

		expr = fragmentExpr

	case ExpressionTypeBinary:
		exprLeft, exprLeftErrors := checkExpr(table, scope, expr.Left)
		exprRight, exprRightErrors := checkExpr(table, scope, expr.Right)

		errors = append(errors, exprLeftErrors...)
		errors = append(errors, exprRightErrors...)

		// replace expressions - a fragment may have been substituted by a new expression with rewritten variables
		expr.Left = exprLeft
		expr.Right = exprRight

		// todo: validate that IS/IS NOT is only with true, false, or null etc
		// and other similar validations

		expr.IsClauseRequired = expr.Left.IsClauseRequired || expr.Right.IsClauseRequired
	case ExpressionTypeLiteral:
		if expr.LiteralType == LiteralTypeFieldName {
			expr.IsClauseRequired = true
			_, e := checkField(table, expr.LiteralFieldName)
			if e.Err != nil {
				errors = append(errors, e)
			}
		} else if expr.LiteralType == LiteralTypeVariable {
			param, e := checkParam(scope, expr.LiteralFieldName)
			if e.Err != nil {
				errors = append(errors, e)
			}
			expr.IsClauseRequired = param.Required
			expr.IsQueryScopedParam = param.IsQueryScoped
			expr.LiteralFieldName = param.GlobalName
			if expr.LiteralFieldName == "" {
				expr.LiteralFieldName = scope.QueryParamToGlobalName[param.Name]
			}
			if expr.LiteralFieldName == "" {
				panic("expected field name to be set")
			}
		}
	default:
		panic("unhandled expression type")
	}

	return expr, errors
}

func checkQuery(schema Schema, fragments []Query, query *Query) []CheckError {
	var errors []CheckError

	scope := Scope{
		Fragments:              fragments,
		QueryParams:            query.Params,
		QueryParamToGlobalName: map[string]string{},
	}

	switch query.StatementType {
	case StatementTypeSelect:

		currentTable := query.Select.From
		tableDef, checkErr := checkTable(schema, currentTable)
		if checkErr.Err != nil {
			errors = append(errors, checkErr)
			// don't continue parsing if table is wrong,
			// otherwise every field will be considered not found
			return errors
		}

		for _, f := range query.Select.Fields {
			if f != "*" {
				_, checkErr = checkField(tableDef, f)
				if checkErr.Err != nil {
					errors = append(errors, checkErr)
				}
			}
		}

		if query.Select.Where.Type > 0 {
			expr, exprErrs := checkExpr(tableDef, scope, &query.Select.Where)
			query.Select.Where = *expr
			errors = append(errors, exprErrs...)
		}

		if len(query.Select.OrderByFields) > 0 {
			for _, f := range query.Select.OrderByFields {
				_, checkErr = checkField(tableDef, f)
				if checkErr.Err != nil {
					errors = append(errors, checkErr)
				}
			}
		}

	default:
		panic("")
	}

	return errors

}

func CheckQueries(schema Schema, queries Queries) []CheckError {
	var errors []CheckError

	fragments := make([]Query, 0, len(queries.Queries))
	for _, q := range queries.Queries {
		if q.IsFragment {
			fragments = append(fragments, q)
		}
	}

	for i, q := range queries.Queries {
		if q.IsFragment {
			continue
		}
		// indexing/pointer because we may modify the query
		errors = append(errors, checkQuery(schema, fragments, &queries.Queries[i])...)
	}

	return errors
}
