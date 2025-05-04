package main

import (
	"fmt"
	"os"
	"strings"
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
	LiteralType        LiteralType
	LiteralNumber      int
	LiteralString      string
	LiteralFieldName   string // this will get rewritten by checker to reference a globally unique name (including across fragments)
	IsQueryScopedParam bool

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

// todo: maybe needs to be more general expression type
// for being able to select from sub tables etc
type SelectStmt struct {
	Fields        []string
	From          string
	Where         Expression
	Limit         int
	OrderByFields []string
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
	//todo: find another way
	panic(err)
}

func (p *QueryParser) EndErrorMode() {
	p.ErrorMode = false

	token := p.PeekToken()

	for token.Type != EOF {
		ident := strings.ToLower(token.Lexeme)
		if token.Type == Identifier && (ident == "query" || ident == "fragment") {
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
		p.AddError(fmt.Errorf("expected identifer type %s, got %s", tokenType, token.Type))
	}

	return token

}

func (p *QueryParser) EatIdentifier(keyword string) Token {
	if p.ErrorMode {
		return Token{}
	}
	token := p.EatTokenOfType(Identifier)

	// todo: see if can remove all to lower calls
	if strings.ToLower(token.Lexeme) != keyword {
		p.AddError(fmt.Errorf("expected identifer type %s", keyword))
	}

	return token
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
func (p *QueryParser) parseOrderBy() []string {
	// todo: check these are field names or something
	var res []string

	// todo: consider turning these into special token type
	_ = p.EatIdentifier("order")
	_ = p.EatIdentifier("by")
	token := p.PeekToken()

	for token.Type != RightBrace {
		token = p.EatTokenOfType(Identifier)
		res = append(res, token.Lexeme)

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

	token := p.EatToken()

	if token.Type == Identifier {
		lowered := strings.ToLower(token.Lexeme)
		if lowered == "null" {
			expr = Expression{
				Type:             ExpressionTypeLiteral,
				LiteralType:      LiteralTypeNull,
				IsClauseRequired: true,
			}
		} else {
			litType := LiteralTypeFieldName
			if p.IsParsingTemplate {
				litType = LiteralTypeVariable
			}

			expr = Expression{
				Type:             ExpressionTypeLiteral,
				LiteralType:      litType,
				LiteralFieldName: token.Lexeme,
				IsClauseRequired: true,
			}
		}
	} else if token.Type == Number {
		number, ok := token.Literal.(NumberLiteral)
		if !ok {
			panic("")
		}
		expr = Expression{
			Type:          ExpressionTypeLiteral,
			LiteralType:   LiteralTypeNumber,
			LiteralNumber: int(number),
		}
	} else if token.Type == LeftBrace {
		token = p.EatTokenOfType(Identifier)

		expr = Expression{
			Type:             ExpressionTypeLiteral,
			LiteralType:      LiteralTypeVariable,
			LiteralFieldName: token.Lexeme,
		}

		token = p.EatTokenOfType(RightBrace)

	} else if token.Type == String {
		expr = Expression{
			Type:          ExpressionTypeLiteral,
			LiteralType:   LiteralTypeString,
			LiteralString: token.Lexeme,
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
	} else if token.Type == Identifier && strings.ToLower(token.Lexeme) == "not" {
		// todo: something better, probably want to know all keyword combinations and how they map to ops
		token = p.PeekToken()
		keyword := strings.ToLower(token.Lexeme)
		if token.Type == Identifier && keyword == "like" {
			opType = OpTypeNotLike
		} else {
			panic("not supported")
		}
	} else if token.Type == Identifier && strings.ToLower(token.Lexeme) == "like" {
		opType = OpTypeLike
	} else if token.Type == Identifier && strings.ToLower(token.Lexeme) == "is" {
		opType = OpTypeIs
		token = p.PeekToken()
		keyword := strings.ToLower(token.Lexeme)
		if token.Type == Identifier && keyword == "not" {
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

	keyword := strings.ToLower(token.Lexeme)

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
			if strings.ToLower(token.Lexeme) != "in" {
				panic("expected in")
			}

			token = p.EatTokenOfType(Identifier)

			expr.ForLoopVarName = token.Lexeme

			token = p.EatTokenOfType(Colon)
			token = p.EatTokenOfType(Identifier)

			lowered := strings.ToLower(token.Lexeme)
			if lowered == "and" {
				expr.ForLoopJoinByOr = false
			} else if lowered == "or" {
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
			elseIsNext := maybeLeftBrace.Type == LeftBrace && maybeElse.Type == Identifier && strings.ToLower(maybeElse.Lexeme) == "else"

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
			keyword = strings.ToLower(token.Lexeme)
			if keyword != "else" {
				break
			}

			token = p.PeekToken()
			if token.Type == RightBrace {
				break
			} else if token.Type == Identifier {
				keyword = strings.ToLower(token.Lexeme)
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

	// todo: tokentype And/Or not used
	for token.Type == Identifier && strings.ToLower(token.Lexeme) == "and" {
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

	for token.Type == Identifier && strings.ToLower(token.Lexeme) == "or" {
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

func (p *QueryParser) parseSelect() SelectStmt {
	var stmt SelectStmt

	token := p.PeekToken()

	// parse select fields
	for !(token.Type == Identifier && strings.ToLower(token.Lexeme) == "from") {
		token = p.EatToken()

		if token.Type == Star {
			stmt.Fields = append(stmt.Fields, "*")
		} else if token.Type == Identifier {
			stmt.Fields = append(stmt.Fields, token.Lexeme)
		} else {
			// todo: report actual error
			panic("unhandled field")
		}

		token = p.PeekToken()

		if token.Type != Comma {
			break
		}

		token = p.EatTokenOfType(Comma)

		token = p.PeekToken()
	}

	token = p.EatIdentifier("from")

	// parse from
	{
		// table name
		token = p.EatTokenOfType(Identifier)
		stmt.From = token.Lexeme

	}

	// optional where clause
	token = p.PeekToken()

	if token.Type == Identifier && strings.ToLower(token.Lexeme) == "where" {

		// todo: try to cut down on error handling that needs to happen during parsing
		// consume the where
		token = p.EatToken()

		expr := p.parseExpression()

		stmt.Where = expr
	}

	token = p.PeekToken()

	if token.Type == Identifier {
		lowered := strings.ToLower(token.Lexeme)
		if lowered == "limit" {
			n := p.parseLimit()
			stmt.Limit = n
		} else if lowered == "order" {
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

			// todo: really clean up this, should be much fewer lines to write this
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

		if strings.ToLower(token.Lexeme) == "select" {
			selectStmt := p.parseSelect()

			query.StatementType = StatementTypeSelect
			query.Select = selectStmt

		} else {
			panic("not supported")
		}
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

		// todo: avoid making copies of token lexemes
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
