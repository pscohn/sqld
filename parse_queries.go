package main

import (
	"fmt"
	"os"
)

// precedence high to low:
// literal/grouping
// multiply/divide
// add/subtract
// comparison (<, >, <=, >=)
// equality (=, !=, like)
// dynamic clause - {foreach}, {include}, {if}
// and
// or

type OpType int

const (
	OpTypeNone OpType = iota
	OpTypeAnd
	OpTypeOr

	OpTypeLess
	OpTypeGreater
	OpTypeLessOrEqual
	OpTypeGreaterOrEqual
	OpTypeEquals
	OpTypeNotEquals

	OpTypeLike
	OpTypeNotLike
	OpTypeIs
	OpTypeIsNot
)

func (opType OpType) String() string {
	var op string
	switch opType {
	case OpTypeAnd:
		op = "AND"
	case OpTypeOr:
		op = "OR"

	case OpTypeLess:
		op = "<"
	case OpTypeGreater:
		op = ">"
	case OpTypeLessOrEqual:
		op = "<="
	case OpTypeGreaterOrEqual:
		op = ">="
	case OpTypeEquals:
		op = "="
	case OpTypeNotEquals:
		op = "!="

	case OpTypeLike:
		op = "LIKE"
	case OpTypeNotLike:
		op = "NOT LIKE"
	case OpTypeIs:
		op = "IS"
	case OpTypeIsNot:
		op = "IS NOT"
	default:
		panic("unhandled op")
	}
	return op
}

type StatementType int

const (
	StatementTypeNone StatementType = iota
	StatementTypeSelect
)

type ExpressionType int

const (
	ExpressionTypeNone ExpressionType = iota
	ExpressionTypeBinary
	ExpressionTypeLiteral
	ExpressionTypeIf
	ExpressionTypeForLoop
	ExpressionTypeFragment
)

type LiteralType int

const (
	LiteralTypeNone LiteralType = iota
	LiteralTypeString
	LiteralTypeNumber
	LiteralTypeFieldName
	LiteralTypeVariable
	LiteralTypeNull
)

type ElseIf struct {
	IfExpr   *Expression
	BodyExpr *Expression
}

type Expression struct {
	Op    OpType
	Left  *Expression
	Right *Expression

	Type ExpressionType

	// literal expression type
	LiteralType         LiteralType
	LiteralNumber       int
	LiteralString       string
	LiteralField        Field
	LiteralVariableName string // this will get rewritten by checker to reference a globally unique name (including across fragments)
	IsQueryScopedParam  bool

	// set to true while checking if any children in its left/right subtrees
	// are also required, or if it's an expression that can be determined as required.
	// an expression is required if references no variables, or if the variable is not nullable.
	// this is initialized during check, not parse phase
	IsClauseRequired bool

	// for-loop expression type (range only)
	ForLoopIteratorName string // the name for each item in the list. this may be rewitten by checker
	ForLoopVarName      string // the list param we're ranging over. this may be rewritten by checker
	ForLoopJoinByOr     bool   // defaults to AND, set to true for OR

	// If expression
	ElseIfs  []ElseIf
	ElseBody *Expression

	// fragment expression type
	FragmentName string
	FragmentArgs []string
}

type Field struct {
	// valid fields:
	// "tablename".id
	// tablename."id" myID
	// tablename.* as myFields (no effect)

	// invalid:
	// * as myFields

	All       bool // *
	Name      string
	TableName string
	Alias     string
}

type JoinType int

const (
	JoinTypeInner JoinType = iota
	JoinTypeCross
	// outer is implied
	JoinTypeLeft
	JoinTypeRight
	JoinTypeFull
)

func (t JoinType) String() string {
	switch t {
	case JoinTypeInner:
		return "INNER JOIN"
	case JoinTypeCross:
		return "CROSS JOIN"
	case JoinTypeLeft:
		return "LEFT JOIN"
	case JoinTypeRight:
		return "RIGHT JOIN"
	case JoinTypeFull:
		return "FULL OUTER JOIN"
	default:
		panic(fmt.Sprintf("unknown join type: %d", t))

	}
}

type Join struct {
	Table      string
	TableAlias string
	JoinType   JoinType
	On         Expression
}

// todo: maybe needs to be more general expression type
// for being able to select from sub tables etc
type SelectStmt struct {
	Fields []Field

	From      string
	FromAlias string

	Joins         []Join
	Where         Expression
	Limit         *int
	OrderByFields []Field // ignores `Alias`
}

type ParamType int

const (
	ParamTypeNone ParamType = iota
	ParamTypeString
	ParamTypeNumber
)

// returns the go type to be used in codegen
func (p ParamType) String() string {
	switch p {
	case ParamTypeString:
		return "string"
	case ParamTypeNumber:
		return "int"
	default:
		panic("unexpected type")
	}
}

type Param struct {
	Name     string
	Type     ParamType
	Required bool
	IsList   bool
	// helps identify to codegen if param comes from input struct or not
	IsQueryScoped bool
	// todo: maybe this replaces IsQueryScoped
	// this provides a globally unique name for params to avoid collisions between fragments with reused names
	GlobalName string
}

// note: fragments only can contain an expression currently,
// so may want to switch to separate type
type Query struct {
	IsFragment bool
	Name       string
	Params     []Param

	// used when IsFragment=false
	StatementType StatementType
	Select        SelectStmt

	// used when IsFragment=true
	FragmentExpression Expression
}

type Queries struct {
	Queries []Query
}

type QueryParser struct {
	Source string
	Result Queries

	Index int

	// controls behavior of expression parser -
	// should assume identifiers are variables instead of field names
	IsParsingTemplate bool

	Scanner    Scanner
	TokenIndex int

	ErrorMode   bool
	ParseErrors []error
}

func NewQueryParser(input string) QueryParser {
	scanner := NewScanner(input)
	p := QueryParser{
		Source:  input,
		Scanner: scanner,
	}
	return p
}

// peek token: scanner returns the current token, but keeps it reserved
// eat token: scanner releases current token

func (p *QueryParser) PeekToken() Token {
	token, err := p.Scanner.PeekToken()
	if err != nil {
		p.AddError(fmt.Errorf("could not get next token"))
	}

	return token
}

func (p *QueryParser) PeekTokenAfter(i int) Token {
	token, err := p.Scanner.PeekTokenAfter(i)
	if err != nil {
		p.AddError(fmt.Errorf("could not get next token"))
	}

	return token
}

func (p *QueryParser) EatToken() Token {
	if p.ErrorMode {
		return Token{}
	}

	token, err := p.Scanner.EatToken()
	if err != nil {
		p.AddError(fmt.Errorf("could not get next token"))

	}

	return token
}

func (p *QueryParser) AddError(err error) {
	// p.ErrorMode = true
	p.ParseErrors = append(p.ParseErrors, err)
	// todo
	panic(err)
}

func (p *QueryParser) EndErrorMode() {
	p.ErrorMode = false

	token := p.PeekToken()

	for token.Type != EOF {
		if token.Type == Identifier && (token.LexemeLowered == "query" || token.LexemeLowered == "fragment") {
			return
		}

		_ = p.EatToken()
		token = p.PeekToken()
	}

}

func (p *QueryParser) EatTokenOfType(tokenType TokenType) Token {
	if p.ErrorMode {
		return Token{}
	}

	token, err := p.Scanner.EatToken()
	if err != nil {
		p.AddError(fmt.Errorf("unable to get next token"))
	}

	if token.Type != tokenType {
		p.AddError(fmt.Errorf("expected token type %s, got %s", tokenType, token.Type))
	}

	return token

}

func (p *QueryParser) EatIdentifier(keyword string) Token {
	if p.ErrorMode {
		return Token{}
	}
	token := p.EatTokenOfType(Identifier)

	if token.LexemeLowered != keyword {
		p.AddError(fmt.Errorf("expected identifier type %s", keyword))
	}

	return token
}

// todo: copied from schema parser, probably just want to combine them
func (p *QueryParser) parseMaybeQuotedName() string {
	token := p.EatToken()
	if token.Type == Identifier {
		return token.Lexeme
	}

	if token.Type == String {
		return token.Literal.String()
	}

	p.AddError(fmt.Errorf("expected name as double quoted string or identifier"))
	return ""
}

// next token is `limit`
func (p *QueryParser) parseLimit() int {
	token := p.EatIdentifier("limit")

	token = p.EatTokenOfType(Number)

	n, ok := token.Literal.(NumberLiteral)
	if !ok {
		panic("expected literal to be NumberLiteral")
	}

	return int(n)
}

// next token is `order`
func (p *QueryParser) parseOrderBy() []Field {
	var res []Field

	_ = p.EatIdentifier(KeywordOrder)
	_ = p.EatIdentifier(KeywordBy)
	token := p.PeekToken()

	for token.Type != RightBrace {

		field := p.parseFieldName()
		res = append(res, field)

		token = p.PeekToken()
		if token.Type != Comma {
			break
		}

		token = p.EatTokenOfType(Comma)
		token = p.PeekToken()
	}

	return res
}

func (p *QueryParser) parseLiteral() Expression {
	var expr Expression

	token := p.PeekToken()

	isNonTemplateSingleQuotedString := token.Type == String && !token.SingleQuoted && !p.IsParsingTemplate
	// todo: will want to check any keyword literal, not just null
	isNonTemplateNonKeywordIdentifier := token.Type == Identifier && token.LexemeLowered != KeywordNull && !p.IsParsingTemplate

	if isNonTemplateSingleQuotedString || isNonTemplateNonKeywordIdentifier {
		field := p.parseFieldName()
		expr = Expression{
			Type:             ExpressionTypeLiteral,
			LiteralType:      LiteralTypeFieldName,
			LiteralField:     field,
			IsClauseRequired: true,
		}
	} else if token.Type == Identifier {
		if token.LexemeLowered == KeywordNull {
			token = p.EatToken()
			expr = Expression{
				Type:             ExpressionTypeLiteral,
				LiteralType:      LiteralTypeNull,
				IsClauseRequired: true,
			}
		} else if p.IsParsingTemplate {
			token = p.EatToken()
			expr = Expression{
				Type:                ExpressionTypeLiteral,
				LiteralType:         LiteralTypeVariable,
				LiteralVariableName: token.Lexeme,
				IsClauseRequired:    true,
			}
		}
	} else if token.Type == Number {
		token = p.EatToken()
		number, ok := token.Literal.(NumberLiteral)
		if !ok {
			panic("expected token to be NumberLiteral")
		}
		expr = Expression{
			Type:          ExpressionTypeLiteral,
			LiteralType:   LiteralTypeNumber,
			LiteralNumber: int(number),
		}
	} else if token.Type == LeftBrace {
		token = p.EatToken()
		token = p.EatTokenOfType(Identifier)

		expr = Expression{
			Type:                ExpressionTypeLiteral,
			LiteralType:         LiteralTypeVariable,
			LiteralVariableName: token.Lexeme,
		}

		token = p.EatTokenOfType(RightBrace)

	} else if token.Type == String {
		token = p.EatToken()
		expr = Expression{
			Type:          ExpressionTypeLiteral,
			LiteralType:   LiteralTypeString,
			LiteralString: token.Literal.String(),
		}

		// todo: handle patterns for like clauses
		// "'%{bioLike}%'"
		// need to put it in a format that we can substitute the variable
		// maybe it's not literal - could be an Expression with prefix/postfix operators and a literal of variable name?
		// could have complex pattern matching that needs to be substituted with a variable

	} else {
		panic("unhandled literal type")
	}

	return expr
}

func (p *QueryParser) parseGrouping() Expression {
	token := p.PeekToken()

	if token.Type == LeftParen {
		token = p.EatToken()

		expr := p.parseExpression()

		token = p.EatTokenOfType(RightParen)

		return expr
	}

	return p.parseLiteral()
}

func (p *QueryParser) parseComparison() Expression {
	left := p.parseGrouping()
	token := p.PeekToken()

	opType := OpTypeEquals

	if token.Type == Equal {
		opType = OpTypeEquals
	} else if token.Type == BangEqual {
		opType = OpTypeNotEquals
	} else if token.Type == Less {
		opType = OpTypeLess
	} else if token.Type == Greater {
		opType = OpTypeGreater
	} else if token.Type == LessEqual {
		opType = OpTypeLessOrEqual
	} else if token.Type == GreaterEqual {
		opType = OpTypeGreaterOrEqual
	} else if token.Type == Identifier && token.LexemeLowered == KeywordNot {
		// todo: something better, probably want to know all keyword combinations and how they map to ops
		token = p.PeekToken()
		if token.Type == Identifier && token.LexemeLowered == KeywordLike {
			opType = OpTypeNotLike
		} else {
			panic("not supported")
		}
	} else if token.Type == Identifier && token.LexemeLowered == KeywordLike {
		opType = OpTypeLike
	} else if token.Type == Identifier && token.LexemeLowered == KeywordIs {
		opType = OpTypeIs
		token = p.PeekToken()
		if token.Type == Identifier && token.LexemeLowered == KeywordNot {
			opType = OpTypeIsNot
		}
	} else {
		return left
	}

	token = p.EatToken()
	right := p.parseGrouping()

	return Expression{
		Type:  ExpressionTypeBinary,
		Op:    opType,
		Left:  &left,
		Right: &right,
	}
}

func (p *QueryParser) parseDynamicClause() Expression {
	token := p.PeekToken()
	if token.Type != LeftBrace {
		// not starting a dynamic clause - passthrough
		return p.parseComparison()
	}

	token = p.PeekTokenAfter(1)

	if token.Type != Identifier {
		return p.parseComparison()
	}

	keyword := token.LexemeLowered

	var expr Expression
	if keyword == "foreach" {
		expr.Type = ExpressionTypeForLoop
		// parse opening
		// 	{foreach query in queries: AND}
		{
			// eat {
			_ = p.EatToken()
			// eat foreach
			_ = p.EatToken()

			token = p.EatTokenOfType(Identifier)

			expr.ForLoopIteratorName = token.Lexeme

			token = p.EatTokenOfType(Identifier)
			if token.LexemeLowered != KeywordIn {
				panic("expected in")
			}

			token = p.EatTokenOfType(Identifier)

			expr.ForLoopVarName = token.Lexeme

			token = p.EatTokenOfType(Colon)
			token = p.EatTokenOfType(Identifier)

			lowered := token.LexemeLowered
			if lowered == KeywordAnd {
				expr.ForLoopJoinByOr = false
			} else if lowered == KeywordOr {
				expr.ForLoopJoinByOr = true
			} else {
				panic("must join for loop with AND or OR")
			}

			token = p.EatTokenOfType(RightBrace)
		}

		// parse body of loop into the Left of the for loop expression.
		// should result in binary expression

		innerExpr := p.parseExpression()
		expr.Left = &innerExpr

		// parse end of loop
		// {end}
		{
			token = p.EatTokenOfType(LeftBrace)
			token = p.EatIdentifier("end")

			token = p.EatTokenOfType(RightBrace)
		}

	} else if keyword == "if" {
		expr.Type = ExpressionTypeIf
		// parse opening
		// 	{if field == nil}
		// eat {
		_ = p.EatToken()
		// eat if
		_ = p.EatToken()

		// parse non-sql expression
		p.IsParsingTemplate = true
		ifExpr := p.parseExpression()
		p.IsParsingTemplate = false
		token = p.EatTokenOfType(RightBrace)

		getMaybeBodyExpression := func() *Expression {
			maybeLeftBrace := p.PeekToken()
			maybeElse := p.PeekTokenAfter(1)
			elseIsNext := maybeLeftBrace.Type == LeftBrace && maybeElse.Type == Identifier && maybeElse.LexemeLowered == KeywordElse

			var bodyExpr *Expression
			if elseIsNext {
				// no inner body
				return nil
			} else {
				innerExpr := p.parseExpression()
				if innerExpr.Type != ExpressionTypeNone {
					bodyExpr = &innerExpr
				}
			}

			return bodyExpr
		}

		bodyResult := getMaybeBodyExpression()

		expr.ElseIfs = append(expr.ElseIfs, ElseIf{
			IfExpr:   &ifExpr,
			BodyExpr: bodyResult,
		})

		// parse optional else-ifs
		for {
			token = p.EatTokenOfType(LeftBrace)
			token = p.EatTokenOfType(Identifier)
			keyword = token.LexemeLowered
			if keyword != "else" {
				break
			}

			token = p.PeekToken()
			if token.Type == RightBrace {
				break
			} else if token.Type == Identifier {
				keyword = token.LexemeLowered
				if keyword == "if" {
					_ = p.EatToken()

					p.IsParsingTemplate = true
					ifExpr := p.parseExpression()
					p.IsParsingTemplate = false

					token = p.EatTokenOfType(RightBrace)

					bodyExpr := getMaybeBodyExpression()

					expr.ElseIfs = append(expr.ElseIfs, ElseIf{
						IfExpr:   &ifExpr,
						BodyExpr: bodyExpr,
					})
				} else {
					p.AddError(fmt.Errorf("expected 'if' or '}' after 'else'"))
				}
			}
		}

		if keyword == "else" {
			token = p.EatToken()
			if token.Type == RightBrace {

				// body of else
				innerExpr := p.parseExpression()

				if innerExpr.Type != ExpressionTypeNone {
					expr.ElseBody = &innerExpr
				}

				_ = p.EatTokenOfType(LeftBrace)
				_ = p.EatIdentifier("end")
				_ = p.EatTokenOfType(RightBrace)

			} else if token.Type == Identifier {

			} else {
				p.AddError(fmt.Errorf("expected 'if' or '}' after 'else'"))
			}

		} else if keyword == "end" {
			token = p.EatTokenOfType(RightBrace)
		} else {
			p.AddError(fmt.Errorf("expected 'else' or 'end'"))
		}
	} else if keyword == "include" {

		// eat {
		_ = p.EatToken()
		// eat include
		_ = p.EatToken()

		token = p.EatTokenOfType(Identifier)
		fragmentName := token.Lexeme

		_ = p.EatTokenOfType(LeftParen)

		token = p.PeekToken()

		// for now args have to be an identifier referencing a variable in scope
		args := []string{}

		i := 0
		for token.Type != RightParen {
			if i > 0 {
				_ = p.EatTokenOfType(Comma)
			}

			token = p.EatTokenOfType(Identifier)
			args = append(args, token.Lexeme)

			token = p.PeekToken()
			i++
		}

		_ = p.EatTokenOfType(RightParen)

		_ = p.EatTokenOfType(RightBrace)

		expr.Type = ExpressionTypeFragment
		expr.FragmentName = fragmentName
		expr.FragmentArgs = args

	} else {
		return p.parseComparison()
	}

	return expr
}

// note: very similar code to parseOr
// pratt-style parser probably cleaner
func (p *QueryParser) parseAnd() Expression {
	left := p.parseDynamicClause()
	expr := &left

	token := p.PeekToken()

	for token.Type == Identifier && token.LexemeLowered == KeywordAnd {
		token = p.EatToken()

		right := p.parseDynamicClause()

		expr = &Expression{
			Type:  ExpressionTypeBinary,
			Op:    OpTypeAnd,
			Left:  expr,
			Right: &right,
		}

		token = p.PeekToken()
	}

	return *expr
}

func (p *QueryParser) parseOr() Expression {
	left := p.parseAnd()
	expr := &left

	token := p.PeekToken()

	for token.Type == Identifier && token.LexemeLowered == KeywordOr {
		token = p.EatToken()

		right := p.parseAnd()

		expr = &Expression{
			Type:  ExpressionTypeBinary,
			Op:    OpTypeOr,
			Left:  expr,
			Right: &right,
		}

		token = p.PeekToken()
	}

	return *expr
}

func (p *QueryParser) parseExpression() Expression {
	var expr *Expression

	left := p.parseOr()
	expr = &left

	return *expr
}

// alias rules:
// - fields: can specify reserved keyword only if paired with "as", or in double quotes
// - tables: cannot use quoted alias, or reserved keywords
// as foo
// as "foo"
// foo
// "foo"
func (p *QueryParser) parseAliasForColumn() string {
	token := p.PeekToken()
	if token.Type == Identifier && token.LexemeLowered == KeywordAs {
		p.EatToken()

		// any identifier or string allowed here
		aliasName := p.parseMaybeQuotedName()
		return aliasName
	}

	// no "as" - only non-reserved keywords, or quoted strings, are considered part of the alias
	token = p.PeekToken()
	if token.Type == String || token.Type == Identifier && !IsReservedKeyword(token.LexemeLowered) {
		aliasName := p.parseMaybeQuotedName()
		return aliasName
	}

	return ""
}

func (p *QueryParser) parseAliasForTable() string {
	token := p.PeekToken()
	if token.Type == Identifier && token.LexemeLowered == KeywordAs {
		p.EatToken()

		token = p.EatToken()
		if IsReservedKeyword(token.LexemeLowered) {
			p.AddError(fmt.Errorf("cannot used reserved word as table alias"))
		}

		aliasName := token.Lexeme
		return aliasName
	}

	// with or without "as", only non-reserved identifiers allowed here
	token = p.PeekToken()
	if token.Type == Identifier && !IsReservedKeyword(token.LexemeLowered) {
		p.EatToken()
		aliasName := token.Lexeme
		return aliasName
	}

	return ""
}

func (p *QueryParser) parseFieldName() Field {
	token := p.PeekToken()
	field := Field{}

	if token.Type == Star {
		_ = p.EatToken()
		field.All = true
		return field
	}

	fieldOrTableName := p.parseMaybeQuotedName()

	token = p.PeekToken()
	if token.Type == Dot {
		// first name is the table name
		// now parse possible star/field name again

		p.EatToken()

		field.TableName = fieldOrTableName

		token = p.PeekToken()
		if token.Type == Star {
			p.EatToken()
			field.All = true
			// don't return - syntactically valid to do authors.* as foo
		} else {
			fieldName := p.parseMaybeQuotedName()
			field.Name = fieldName
		}

	} else {
		field.Name = fieldOrTableName
	}

	return field
}

func (p *QueryParser) parseFieldNameWithAlias() Field {
	field := p.parseFieldName()
	aliasName := p.parseAliasForColumn()
	field.Alias = aliasName
	return field
}

func (p *QueryParser) parseJoin() []Join {

	joins := []Join{}

	parseOuterJoin := func(t2, t3 Token) {
		_ = p.EatToken()
		if t2.IsKeyword(KeywordOuter) {
			_ = p.EatToken()

			t := p.EatToken()
			if !t.IsKeyword(KeywordJoin) {
				p.AddError(fmt.Errorf("expected 'join' after 'outer"))
			}
		} else if t2.IsKeyword(KeywordJoin) {
			_ = p.EatToken()
		} else {
			p.AddError(fmt.Errorf("expected 'outer' or 'join'"))
		}
	}

	for p.isJoin() {

		// parse join type
		joinType := JoinTypeInner

		token := p.PeekToken()
		token2 := p.PeekTokenAfter(1)
		token3 := p.PeekTokenAfter(2)

		if token.Type != Identifier {
			p.AddError(fmt.Errorf("expected identifier to start join"))
		}

		switch token.LexemeLowered {
		case KeywordJoin:
			_ = p.EatToken()
			joinType = JoinTypeInner
		case KeywordInner:
			if !token2.IsKeyword(KeywordJoin) {
				p.AddError(fmt.Errorf("expected 'join' after 'inner'"))
			}
			_ = p.EatToken()
			_ = p.EatToken()
			joinType = JoinTypeInner
		case KeywordCross:
			_ = p.EatToken()
			joinType = JoinTypeCross
		case KeywordLeft:
			joinType = JoinTypeLeft
			parseOuterJoin(token2, token3)
		case KeywordRight:
			joinType = JoinTypeLeft
			parseOuterJoin(token2, token3)
		case KeywordFull:
			joinType = JoinTypeFull
			parseOuterJoin(token2, token3)
		}

		// parse table with alias
		token = p.EatTokenOfType(Identifier)
		table := token.Lexeme
		alias := p.parseAliasForTable()

		// parse ON expression.
		// not required with cross, natural, or using - unsupported for now

		p.EatIdentifier(KeywordOn)
		expr := p.parseExpression()

		joins = append(joins, Join{
			Table:      table,
			TableAlias: alias,
			JoinType:   joinType,
			On:         expr,
		})

	}

	return joins
}

func (p *QueryParser) isJoin() bool {
	token := p.PeekToken()
	token2 := p.PeekTokenAfter(1)
	token3 := p.PeekTokenAfter(2)

	// simplified check, will actually try to properly parse if join seems to be here:
	if token.IsKeyword(KeywordJoin) || token2.IsKeyword(KeywordJoin) || token3.IsKeyword(KeywordJoin) {
		return true
	}

	// possibly do this more precise check
	// if token.IsKeyword(KeywordInner, KeywordLeft, KeywordRight, KeywordCross) &&
	// 	(token2.IsKeyword(KeywordJoin) || (token2.IsKeyword(KeywordOuter) && token3.IsKeyword(KeywordJoin))) {
	// 	return true
	// }

	return false
}

func (p *QueryParser) parseSelect() SelectStmt {
	var stmt SelectStmt

	token := p.PeekToken()

	// parse select fields
	for !(token.Type == Identifier && token.LexemeLowered == KeywordFrom) {
		field := p.parseFieldNameWithAlias()
		stmt.Fields = append(stmt.Fields, field)

		token = p.PeekToken()

		if token.Type != Comma {
			break
		}

		token = p.EatTokenOfType(Comma)

		token = p.PeekToken()
	}

	token = p.EatIdentifier(KeywordFrom)

	// parse from
	{
		// table name
		token = p.EatTokenOfType(Identifier)
		stmt.From = token.Lexeme
		stmt.FromAlias = p.parseAliasForTable()
	}

	// optional join clause
	stmt.Joins = p.parseJoin()

	// optional where clause
	token = p.PeekToken()

	if token.IsKeyword(KeywordWhere) {

		// consume the where
		token = p.EatToken()

		expr := p.parseExpression()

		stmt.Where = expr
	}

	token = p.PeekToken()

	if token.Type == Identifier {
		if token.LexemeLowered == KeywordLimit {
			n := p.parseLimit()
			stmt.Limit = &n
		} else if token.LexemeLowered == KeywordOrder {
			sortFields := p.parseOrderBy()
			stmt.OrderByFields = sortFields
		}
	}

	return stmt
}

func (p *QueryParser) parseQuery(isFragment bool) {
	var query Query
	query.IsFragment = isFragment

	// query or fragment name
	token := p.EatTokenOfType(Identifier)

	query.Name = token.Lexeme

	token = p.PeekToken()

	// optional params
	if token.Type == LeftParen {
		token = p.EatTokenOfType(LeftParen)

		token = p.PeekToken()
		params := []Param{}

		// (id: string?, foo: number, bar: number!)
		for token.Type != RightParen {
			param := Param{}
			param.IsQueryScoped = true
			token = p.EatTokenOfType(Identifier)
			param.Name = token.Lexeme
			if !isFragment {
				param.GlobalName = "input." + param.Name
			}

			token = p.EatTokenOfType(Colon)

			token = p.PeekToken()

			if token.Type == LeftBracket {
				param.IsList = true

				_ = p.EatToken()

				// only write to `token` for the identifier since it's checked after
				// consuming the right bracket
				token = p.EatTokenOfType(Identifier)

				_ = p.EatTokenOfType(RightBracket)
			} else {
				token = p.EatTokenOfType(Identifier)
			}

			switch token.Lexeme {
			case "string":
				param.Type = ParamTypeString
			case "int":
				param.Type = ParamTypeNumber
			default:
				p.AddError(fmt.Errorf("unrecognized type: %s", token.Lexeme))
			}

			token = p.PeekToken()

			if token.Type == Bang {
				token = p.EatToken()
				param.Required = true
			} else if token.Type == QuestionMark {
				token = p.EatToken()
				param.Required = false
			} else {
				param.Required = true
			}

			params = append(params, param)

			token = p.PeekToken()

			if token.Type == Comma {
				token = p.EatToken()
			}

			token = p.PeekToken()
		}

		query.Params = params

		token = p.EatTokenOfType(RightParen)
	}

	token = p.EatTokenOfType(LeftBrace)

	if isFragment {
		expr := p.parseExpression()
		query.FragmentExpression = expr

	} else {
		token = p.EatTokenOfType(Identifier)

		if token.LexemeLowered == KeywordSelect {
			selectStmt := p.parseSelect()

			query.StatementType = StatementTypeSelect
			query.Select = selectStmt

		} else {
			panic("not supported")
		}
	}

	token = p.PeekToken()
	if token.Type == Semicolon {
		_ = p.EatToken()
	}

	token = p.EatTokenOfType(RightBrace)

	p.Result.Queries = append(p.Result.Queries, query)
}

func (p *QueryParser) Parse() {

	for p.Scanner.HasNextToken() {
		// todo: finish implementing, there are probably infinite loops because tokens don't advance on error
		// p.EndErrorMode()

		token, err := p.Scanner.EatToken()
		if err != nil {
			panic("")

		}

		if token.Type != Identifier {
			// error
			panic("")
		}

		queryType := token.Lexeme
		if queryType != "query" && queryType != "fragment" {
			panic("")
		}

		p.parseQuery(queryType == "fragment")
	}
}

func parseQueries(filename string) (Queries, error) {

	text, err := os.ReadFile(filename)
	if err != nil {
		return Queries{}, err
	}

	parser := NewQueryParser(string(text))

	// uncomment this to scan and print out all tokens for scan debugging:
	// allTokens, err := schemaParser.Scanner.ScanTokens()
	// fmt.Println(err)
	// fmt.Println(allTokens)

	parser.Parse()

	return parser.Result, nil

}
