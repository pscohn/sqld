package main

import (
	"fmt"
	"os"
	"strings"
)

type TableFieldType int

const (
	TableFieldTypeNone TableFieldType = iota
	TableFieldTypeBigSerial
	TableFieldTypeText
)

func (t TableFieldType) String() string {
	switch t {
	case TableFieldTypeNone:
		return "(unknown)"
	case TableFieldTypeBigSerial:
		return "BIG SERIAL"
	case TableFieldTypeText:
		return "TEXT"
	}
	panic(fmt.Sprintf("table field type %d not handled", t))
}

type TableField struct {
	Name       string
	Type       TableFieldType
	PrimaryKey bool
	NotNull    bool
}

type Table struct {
	Schema string // optional
	Name   string
	Fields []TableField
}

type Schema struct {
	Tables []Table
}

type SchemaParser struct {
	Source string
	Result Schema

	Index int

	Scanner    Scanner
	TokenIndex int

	ParseErrors []error
}

func NewSchemaParser(input string) SchemaParser {
	scanner := NewScanner(input)
	p := SchemaParser{
		Source:  input,
		Scanner: scanner,
	}
	return p
}

// peek token: scanner returns the current token, but keeps it reserved
// eat token: scanner releases current token

func (p *SchemaParser) AddError(err error) Token {
	p.ParseErrors = append(p.ParseErrors, err)
	// todo
	panic(err)
}

func (p *SchemaParser) PeekToken() Token {
	token, err := p.Scanner.PeekToken()
	if err != nil {
		p.AddError(err)
	}

	return token
}

func (p *SchemaParser) EatToken() Token {
	token, err := p.Scanner.EatToken()
	if err != nil {
		p.AddError(err)
	}

	return token
}

func (p *SchemaParser) EatTokenOfType(tokenType TokenType) Token {
	token, err := p.Scanner.EatToken()
	if err != nil {
		p.AddError(err)
	}

	if token.Type != tokenType {
		p.AddError(fmt.Errorf("expected token type %s, got %s", tokenType, token.Type))
	}

	return token
}

func (p *SchemaParser) TableFieldTypeFromString(tableType string) TableFieldType {
	s := strings.ToLower(tableType)

	switch s {
	case "bigserial":
		return TableFieldTypeBigSerial
	case "text":
		return TableFieldTypeText
	default:
		// allow parsing schemas even if we don't recognize all types
		return TableFieldTypeNone
	}
}

func (p *SchemaParser) parseMaybeQuotedName() string {
	token := p.EatToken()
	if token.Type == Identifier {
		return token.Lexeme
	}

	if token.Type == String {
		return token.Lexeme
	}

	p.AddError(fmt.Errorf("expected name as double quoted string or identifier"))
	return ""
}

func (p *SchemaParser) parseTableField() TableField {
	// eg:
	// id BIGSERIAL PRIMARY KEY

	var field TableField

	field.Name = p.parseMaybeQuotedName()

	// type
	token := p.EatTokenOfType(Identifier)

	field.Type = p.TableFieldTypeFromString(token.Lexeme)

	// parse options

	token = p.PeekToken()

	// note: not parsing all possible options here.
	// just common options or ones we care about.
	unclosedParenCount := 0
	for unclosedParenCount > 0 || (token.Type != Comma && token.Type != RightParen) {
		token = p.EatToken()

		if token.Type == LeftParen {
			unclosedParenCount++
		} else if unclosedParenCount > 0 && token.Type == RightParen {
			unclosedParenCount--
		}
		if token.Type != Identifier {
			token = p.PeekToken()
			continue
		}

		switch token.LexemeLowered {
		case KeywordPrimary:
			token = p.EatTokenOfType(Identifier)

			if token.LexemeLowered == KeywordKey {
				field.PrimaryKey = true
			} else {
				// not supported
			}
		case KeywordNull:
			field.NotNull = false
		case KeywordNot:
			token = p.EatTokenOfType(Identifier)

			if token.LexemeLowered == KeywordNull {
				field.NotNull = true
			} else {
				// not supported
			}
		}

		// reset token for next iteration
		token = p.PeekToken()
	}

	// comma, or may not be a trailing comma
	if token.Type == Comma {
		token = p.EatToken()
	}

	return field

}

func (p *SchemaParser) parseTableSchemaAndName() (string, string) {
	var schemaName, tableName string

	name1 := p.parseMaybeQuotedName()

	token := p.PeekToken()
	if token.Type == Dot {
		// eat the .
		p.EatToken()

		schemaName = name1
		tableName = p.parseMaybeQuotedName()
	} else {
		tableName = name1
	}

	return schemaName, tableName
}

func (p *SchemaParser) parseTable() {
	var table Table

	table.Schema, table.Name = p.parseTableSchemaAndName()

	token := p.EatTokenOfType(LeftParen)

	token = p.PeekToken()

	for token.Type != RightParen {
		field := p.parseTableField()
		table.Fields = append(table.Fields, field)
		token = p.PeekToken()
	}

	_ = p.EatTokenOfType(RightParen)
	_ = p.EatTokenOfType(Semicolon)

	p.Result.Tables = append(p.Result.Tables, table)
}

func (p *SchemaParser) Parse() {
	for p.Scanner.HasNextToken() {
		// skip all statements until we find a "create table"

		token := p.PeekToken()
		token2, err := p.Scanner.PeekTokenAfter(1)
		if err != nil {
			panic(err)
		}

		isCreate := token.Type == Identifier && token.LexemeLowered == KeywordCreate
		isTable := token2.Type == Identifier && token2.LexemeLowered == KeywordTable

		if !(isCreate && isTable) {
			// statement we don't parse, skip until semicolon
			for token.Type != Semicolon && p.Scanner.HasNextToken() {
				token = p.EatToken()
			}
			continue
		}

		token = p.EatToken()
		token = p.EatToken()
		p.parseTable()
	}
}

func parseSchema(filename string) (Schema, error) {
	schemaText, err := os.ReadFile(filename)
	if err != nil {
		return Schema{}, err
	}

	schemaParser := NewSchemaParser(string(schemaText))
	// allTokens, err := schemaParser.Scanner.ScanTokens()
	// fmt.Println(err)
	// fmt.Println(allTokens)
	schemaParser.Parse()

	return schemaParser.Result, nil

}
