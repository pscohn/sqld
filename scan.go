package main

import (
	"fmt"
	"strconv"
	"strings"
)

// todo: there is some stuff in here copied from another scanner that needs to be removed

type TokenType int

const (
	// single character tokens
	LeftParen TokenType = iota
	RightParen
	LeftBrace    // {
	RightBrace   // }
	LeftBracket  // [
	RightBracket // ]
	Comma
	Dot
	Minus
	Plus
	Semicolon
	Slash
	Star
	Colon

	// one or two character tokens
	QuestionMark
	Bang
	BangEqual
	Equal
	EqualEqual
	Greater
	GreaterEqual
	Less
	LessEqual

	Identifier
	String
	Number

	And
	Class
	Else
	False
	Fun
	For
	If
	Nil
	Or
	Print
	Stdin
	Return
	Super
	This
	True

	TypeU32
	TypeBool
	TypeString

	EOF
)

func (t TokenType) String() string {
	switch t {
	case LeftParen:
		return "LeftParen"
	case RightParen:
		return "RightParen"
	case LeftBrace:
		return "LeftBrace"
	case RightBrace:
		return "RightBrace"
	case LeftBracket:
		return "LeftBracket"
	case RightBracket:
		return "RightBracket"
	case Comma:
		return "Comma"
	case Dot:
		return "Dot"
	case Minus:
		return "Minus"
	case Plus:
		return "Plus"
	case Semicolon:
		return "Semicolon"
	case Slash:
		return "Slash"
	case Star:
		return "Star"
	case Colon:
		return "Colon"
	case QuestionMark:
		return "QuestionMark"
	case Bang:
		return "Bang"
	case BangEqual:
		return "BangEqual"
	case Equal:
		return "Equal"
	case EqualEqual:
		return "EqualEqual"
	case Greater:
		return "Greater"
	case GreaterEqual:
		return "GreaterEqual"
	case Less:
		return "Less"
	case LessEqual:
		return "LessEqual"
	case Identifier:
		return "Identifier"
	case String:
		return "String"
	case Number:
		return "Number"
	case And:
		return "And"
	case Class:
		return "Class"
	case Else:
		return "Else"
	case False:
		return "False"
	case Fun:
		return "Fun"
	case For:
		return "For"
	case If:
		return "If"
	case Nil:
		return "Nil"
	case Or:
		return "Or"
	case Print:
		return "Print"
	case Stdin:
		return "Stdin"
	case Return:
		return "Return"
	case Super:
		return "Super"
	case This:
		return "This"
	case True:
		return "True"
	case TypeU32:
		return "TypeU32"
	case TypeBool:
		return "TypeBool"
	case TypeString:
		return "TypeString"
	case EOF:
		return "EOF"
	default:
		return "Unknown"
	}
}

// todo
var keywords = map[string]TokenType{
	// "and":    And,
	// "or":     Or,
}

type Literal interface {
	IsLiteral()
	String() string
}

type BoolLiteral bool

func (BoolLiteral) IsLiteral() {}
func (s BoolLiteral) String() string {
	if s {
		return "true"
	}
	return "false"
}

type StringLiteral string

func (StringLiteral) IsLiteral() {}
func (s StringLiteral) String() string {
	return string(s)
}

// todo: separate number types as needed
type NumberLiteral float64

func (NumberLiteral) IsLiteral() {}
func (n NumberLiteral) String() string {
	s := strconv.FormatFloat(float64(n), 'E', -1, 64)
	return string(s)
}

type Token struct {
	Type    TokenType
	Lexeme  string
	Literal Literal
	Line    int
}

func (t Token) String() string {
	return t.Type.String() + " " + t.Lexeme + " " + t.Literal.String()
}

const RingBufferSize = 5

type Scanner struct {
	Source string

	// note: this is probably overkill, but will wait until parser is more complete.
	// so far only 1-2 token lookahead is needed
	TokenRingBuffer [RingBufferSize]Token
	BufferSize      int
	BufferStart     int

	start   int
	current int
	line    int
}

func NewScanner(source string) Scanner {
	return Scanner{
		Source: source,
		line:   1,
	}
}

// todo: redo this
func (s *Scanner) HasNextToken() bool {
	if s.BufferSize > 0 {
		return true
	}

	if s.isAtEnd() {
		return false
	}

	_, err := s.PeekToken()
	if err != nil {
		panic("do something")
	}

	if s.BufferSize == 0 {
		return false
	}

	return true
}

// todo: peek token returns a token, but there may not actually be one if called at end of file
func (s *Scanner) PeekToken() (Token, error) {
	if s.BufferSize == 0 {
		s.scanToken()
	}

	return s.TokenRingBuffer[s.BufferStart], nil

}

// lookahead by `n` tokens after the current token
func (s *Scanner) PeekTokenAfter(n int) (Token, error) {
	for s.BufferSize < n+1 {
		s.scanToken()
	}

	return s.TokenRingBuffer[(s.BufferStart+n)%RingBufferSize], nil
}

func (s *Scanner) EatToken() (Token, error) {
	if s.BufferSize == 0 {
		s.scanToken()
	}

	if s.BufferSize <= 0 {
		return Token{}, fmt.Errorf("no tokens")
	}

	token := s.TokenRingBuffer[s.BufferStart]
	if s.BufferSize > 1 {
		// only need to update start if there's something else in the buffer.
		// otherwise reuse the start position
		s.BufferStart = (s.BufferStart + 1) % RingBufferSize
	}
	s.BufferSize--
	return token, nil
}

// used for debugging only
func (s *Scanner) ScanTokens() ([]Token, error) {
	var tokens []Token
	for !s.isAtEnd() {
		err := s.scanToken()
		if err != nil {
			return nil, err
		}

		token, err := s.EatToken()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	tokens = append(tokens, Token{
		Type: EOF,
		Line: s.line,
	})

	return tokens, nil
}

func (s Scanner) isAtEnd() bool {
	return s.current >= len(s.Source)
}

func isWhitespace(c string) bool {
	switch c {
	case " ", "\t", "\r", "\n":
		return true
	}
	return false
}

// scanToken adds the next token to the ring buffer, skipping whitespace as needed.
func (s *Scanner) scanToken() error {
	c := s.advance()
	for isWhitespace(c) {
		if c == "\n" {
			s.line++
		}

		// todo: cleaner way to handle this - errors if ends in newline and tab
		if s.isAtEnd() {
			return nil
		}

		c = s.advance()
	}

	// already incremented current, so start is one less
	// todo: don't love this
	s.start = s.current - 1

	switch c {

	// one character
	case "(":
		s.addToken(LeftParen)
	case ")":
		s.addToken(RightParen)
	case "{":
		s.addToken(LeftBrace)
	case "}":
		s.addToken(RightBrace)
	case "[":
		s.addToken(LeftBracket)
	case "]":
		s.addToken(RightBracket)
	case ",":
		s.addToken(Comma)
	case ".":
		s.addToken(Dot)
	case "-":
		s.addToken(Minus)
	case "+":
		s.addToken(Plus)
	case ";":
		s.addToken(Semicolon)
	case "*":
		s.addToken(Star)
	case ":":
		s.addToken(Colon)

	// one/two characters
	case "?":
		s.addToken(QuestionMark)
	case "!":
		t := Bang
		if s.match('=') {
			t = BangEqual
		}
		s.addToken(t)
	case "=":
		t := Equal
		if s.match('=') {
			t = EqualEqual
		}
		s.addToken(t)
	case "<":
		t := Less
		if s.match('=') {
			t = LessEqual
		} else if s.match('>') {
			// rewrite to !=
			t = BangEqual
		}
		s.addToken(t)
	case ">":
		t := Greater
		if s.match('=') {
			t = GreaterEqual
		}
		s.addToken(t)

	// division/comment
	case "/":
		if s.match('/') {
			// advance until end of line
			for s.peek() != "\n" && !s.isAtEnd() {
				s.advance()
			}
		} else {
			s.addToken(Slash)
		}

	// 	// skip whitespace
	// case " ", "\t", "\r":
	// case "\n":
	// 	s.line++

	// literals
	case "\"":
		err := s.string("\"")
		if err != nil {
			return err
		}
	case "'":
		err := s.string("'")
		if err != nil {
			return err
		}

	default:
		if isDigit(c) {
			err := s.number()
			if err != nil {
				return err
			}
		} else if isAlpha(c) {
			s.identifier()
		} else {
			return fmt.Errorf("line %d: Unexpected character: %s", s.line, c)
		}
	}
	return nil
}

func isAlpha(c string) bool {
	return (c >= "a" && c <= "z") || (c >= "A" && c <= "Z") || c == "_"
}

func isAlphaNumeric(c string) bool {
	return isAlpha(c) || isDigit(c)
}

func isDigit(c string) bool {
	return c >= "0" && c <= "9"
}

func (s *Scanner) identifier() {
	for isAlphaNumeric(s.peek()) {
		s.advance()
	}

	text := strings.ToLower(s.Source[s.start:s.current])
	if t, ok := keywords[text]; ok {
		s.addToken(t)
		return
	}

	s.addToken(Identifier)
}

func (s *Scanner) number() error {
	for isDigit(s.peek()) {
		s.advance()
	}

	// look for fractional part
	if s.peek() == "." && isDigit(s.peekNext()) {
		// consume "."
		s.advance()

		for isDigit(s.peek()) {
			s.advance()
		}
	}

	res := s.Source[s.start:s.current]

	num, err := strconv.ParseFloat(res, 64)
	if err != nil {
		return fmt.Errorf("line %d: Error parsing float: %s", s.line, err.Error())
	}

	s.addTokenLiteral(Number, NumberLiteral(num))
	return nil
}

func (s *Scanner) string(endChar string) error {
	for s.peek() != endChar && !s.isAtEnd() {
		if s.peek() == "\n" {
			s.line++
		}
		s.advance()
	}

	if s.isAtEnd() {
		return fmt.Errorf("line %d: Unterminated string", s.line)
	}

	// consume closing "
	s.advance()

	// trim surrounding quotes
	value := s.Source[s.start+1 : s.current-1]

	s.addTokenLiteral(String, StringLiteral(value))

	return nil
}

func (s *Scanner) peek() string {
	if s.isAtEnd() {
		return ""
	}
	return string(s.Source[s.current])
}

func (s *Scanner) peekNext() string {
	if s.current+1 >= len(s.Source) {
		return ""
	}
	return string(s.Source[s.current+1])
}

func (s *Scanner) match(expected rune) bool {
	if s.isAtEnd() {
		return false
	}

	if s.Source[s.current] != byte(expected) {
		return false
	}

	s.current++
	return true
}

func (s *Scanner) advance() string {
	curr := string(s.Source[s.current])
	s.current++
	return curr
}

func (s *Scanner) addToken(t TokenType) {
	s.addTokenLiteral(t, nil)
}

func (s *Scanner) addTokenLiteral(t TokenType, literal Literal) {
	if s.BufferSize >= RingBufferSize {
		panic("ring buffer full")
	}

	text := s.Source[s.start:s.current]
	token := Token{
		Type:    t,
		Lexeme:  text,
		Line:    s.line,
		Literal: literal,
	}

	index := (s.BufferStart + s.BufferSize) % RingBufferSize
	s.TokenRingBuffer[index] = token
	s.BufferSize++
}
