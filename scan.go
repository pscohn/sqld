package main

import (
	"fmt"
	"strconv"
	"strings"
)

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
	case EOF:
		return "EOF"
	default:
		return "Unknown"
	}
}

// it would be nice to use known keyword tokens for some identifiers, but
// even reserved words can be used as column labels, so it may not be possible to know
// until parsing: https://www.postgresql.org/docs/17/sql-keywords-appendix.html
type Keyword = string

const (
	KeywordCreate Keyword = "create"
	KeywordTable  Keyword = "table"

	KeywordSelect Keyword = "select"
	KeywordFrom   Keyword = "from"
	KeywordWhere  Keyword = "where"
	KeywordIn     Keyword = "in"
	KeywordLimit  Keyword = "limit"
	KeywordOrder  Keyword = "order"
	KeywordBy     Keyword = "by"

	KeywordAnd     Keyword = "and"
	KeywordOr      Keyword = "or"
	KeywordFor     Keyword = "for"
	KeywordIf      Keyword = "if"
	KeywordElse    Keyword = "else"
	KeywordNull    Keyword = "null"
	KeywordTrue    Keyword = "true"
	KeywordFalse   Keyword = "false"
	KeywordNot     Keyword = "not"
	KeywordLike    Keyword = "like"
	KeywordIs      Keyword = "is"
	KeywordPrimary Keyword = "primary"
	KeywordKey     Keyword = "key"
	KeywordAs      Keyword = "as"

	KeywordJoin  Keyword = "join"
	KeywordOn    Keyword = "on"
	KeywordInner Keyword = "inner"
	KeywordLeft  Keyword = "left"
	KeywordRight Keyword = "right"
	KeywordOuter Keyword = "outer"
	KeywordCross Keyword = "cross"
	KeywordFull  Keyword = "full"
)

// todo: flesh out list
func IsReservedKeyword(k Keyword) bool {
	switch k {
	case
		KeywordFrom,
		KeywordWhere,
		KeywordLimit,
		KeywordJoin,
		KeywordOn,
		KeywordInner,
		KeywordOuter,
		KeywordCross,
		KeywordFull,
		KeywordLeft,
		KeywordRight:
		return true
	}
	return false

}

// todo: I don't really like how literals are handled at the moment.
// for one thing, have to call through interface to get string value, because
// lexeme includes quotes. and using an interface here is inconsistent with the
// general approach of wide structs.
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
	Type         TokenType
	Lexeme       string
	Literal      Literal
	SingleQuoted bool // for strings
	Line         int

	// This is populated for all identifiers, and is needed
	// to check if the input is a keyword.
	LexemeLowered string
}

func (t Token) String() string {
	return t.Type.String() + " " + t.Lexeme + " " + t.Literal.String()
}

func (t Token) IsKeyword(keywords ...string) bool {
	if t.Type != Identifier {
		return false
	}

	for _, k := range keywords {
		if t.LexemeLowered == k {
			return true
		}
	}

	return false
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
	if s.isAtEnd() {
		return false
	}

	t, err := s.PeekToken()
	if err != nil {
		panic("do something")
	}

	if t.Type == EOF {
		return false
	}

	return true
}

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
	if s.isAtEnd() {
		s.addToken(EOF)
		return nil
	}

	c := s.advance()
	for isWhitespace(c) {
		if c == "\n" {
			s.line++
		}

		// todo: cleaner way to handle this - errors if ends in newline and tab
		if s.isAtEnd() {
			s.addToken(EOF)
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
		if s.match('-') {
			// advance until end of line
			for s.peek() != "\n" && !s.isAtEnd() {
				s.advance()
			}
		} else {
			s.addToken(Minus)
		}
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

	s.addTokenLiteral(Number, NumberLiteral(num), false)
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

	isSingleQuoted := endChar == "'"

	s.addTokenLiteral(String, StringLiteral(value), isSingleQuoted)

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
	s.addTokenLiteral(t, nil, false)
}

// todo: better design for singleQuoted
func (s *Scanner) addTokenLiteral(t TokenType, literal Literal, singleQuoted bool) {
	if s.BufferSize >= RingBufferSize {
		panic("ring buffer full")
	}

	text := s.Source[s.start:s.current]
	token := Token{
		Type:         t,
		Lexeme:       text,
		Line:         s.line,
		Literal:      literal,
		SingleQuoted: singleQuoted,
	}

	if t == Identifier {
		token.LexemeLowered = strings.ToLower(text)
	}

	index := (s.BufferStart + s.BufferSize) % RingBufferSize
	s.TokenRingBuffer[index] = token
	s.BufferSize++
}
