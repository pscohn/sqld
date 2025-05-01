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
	Name string
	// todo: preallocate/max buffer?
	Fields []TableField
}

type Schema struct {
	// todo: preallocate/max buffer?
	Tables []Table
}

type SchemaParser struct {
	Source string
	Result Schema

	Index int

	Scanner    Scanner
	TokenIndex int
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

	}

	if token.Type != tokenType {
		p.AddError(fmt.Errorf("expected token type"))
	}

	return token
}

func (p *SchemaParser) TableFieldTypeFromString(tableType string) TableFieldType {
	// todo: consider better way to sanitize user input than calling ToLower all over the place

	s := strings.ToLower(tableType)

	switch s {
	case "bigserial":
		return TableFieldTypeBigSerial
	case "text":
		return TableFieldTypeText
	}

	p.AddError(fmt.Errorf("unknown field type: %s", s))
	return 0
}

func (p *SchemaParser) parseTableField() TableField {
	// eg:
	// id BIGSERIAL PRIMARY KEY

	var field TableField

	// field name
	token := p.EatTokenOfType(Identifier)

	field.Name = token.Lexeme

	// type
	token = p.EatTokenOfType(Identifier)

	field.Type = p.TableFieldTypeFromString(token.Lexeme)

	// parse options

	token = p.PeekToken()

	for token.Type != Comma && token.Type != RightParen {
		token = p.EatTokenOfType(Identifier)

		// todo: look at efficient lookups
		lower := strings.ToLower(token.Lexeme)
		switch lower {
		case "primary":
			token = p.EatTokenOfType(Identifier)

			if strings.ToLower(token.Lexeme) == "key" {
				field.PrimaryKey = true
			} else {
				// todo
				panic("not supported")
			}
		case "null":
			field.NotNull = false
		case "not":
			token = p.EatTokenOfType(Identifier)

			if strings.ToLower(token.Lexeme) == "null" {
				field.NotNull = true
			} else {
				// todo
				panic("not supported")
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

func (p *SchemaParser) parseTable() {
	var table Table
	// table name
	token := p.EatTokenOfType(Identifier)

	table.Name = token.Lexeme

	token = p.EatTokenOfType(LeftParen)

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
		token := p.EatToken()

		if token.Type != Identifier && strings.ToLower(token.Lexeme) != "create" {
			// todo
			panic("")
		}

		token = p.EatToken()
		if token.Type != Identifier && strings.ToLower(token.Lexeme) != "table" {
			// todo
			panic("")
		}

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
