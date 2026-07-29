package native

import (
	"fmt"
	"unicode"
)

type tokenKind int

const (
	tokenEOF tokenKind = iota
	tokenIdent
	tokenInt
	tokenString
	tokenLBrace
	tokenRBrace
	tokenLBracket
	tokenRBracket
	tokenLParen
	tokenRParen
	tokenColon
	tokenSemi
	tokenAssign
	tokenDot
	tokenQuote
	tokenPlus
	tokenMinus
	tokenTimes
	tokenDivide
)

type token struct {
	kind tokenKind
	lit  string
	line int
	col  int
}

type lexer struct {
	src  []rune
	pos  int
	line int
	col  int
}

func newLexer(s string) *lexer {
	return &lexer{src: []rune(s), line: 1, col: 1}
}

func (l *lexer) nextToken() (token, error) {
	if err := l.skipWhitespaceAndComments(); err != nil {
		return token{}, err
	}
	if l.eof() {
		return token{kind: tokenEOF, line: l.line, col: l.col}, nil
	}

	ch := l.peek()
	startLine, startCol := l.line, l.col

	if isIdentStart(ch) {
		lit := l.readIdent()
		return token{kind: tokenIdent, lit: lit, line: startLine, col: startCol}, nil
	}
	if unicode.IsDigit(ch) {
		lit := l.readInt()
		return token{kind: tokenInt, lit: lit, line: startLine, col: startCol}, nil
	}
	if ch == '"' {
		lit, err := l.readString()
		if err != nil {
			return token{}, err
		}
		return token{kind: tokenString, lit: lit, line: startLine, col: startCol}, nil
	}

	l.advance()
	switch ch {
	case '{':
		return token{kind: tokenLBrace, lit: "{", line: startLine, col: startCol}, nil
	case '}':
		return token{kind: tokenRBrace, lit: "}", line: startLine, col: startCol}, nil
	case '[':
		return token{kind: tokenLBracket, lit: "[", line: startLine, col: startCol}, nil
	case ']':
		return token{kind: tokenRBracket, lit: "]", line: startLine, col: startCol}, nil
	case '(':
		return token{kind: tokenLParen, lit: "(", line: startLine, col: startCol}, nil
	case ')':
		return token{kind: tokenRParen, lit: ")", line: startLine, col: startCol}, nil
	case ':':
		return token{kind: tokenColon, lit: ":", line: startLine, col: startCol}, nil
	case ';':
		return token{kind: tokenSemi, lit: ";", line: startLine, col: startCol}, nil
	case '=':
		return token{kind: tokenAssign, lit: "=", line: startLine, col: startCol}, nil
	case '.':
		return token{kind: tokenDot, lit: ".", line: startLine, col: startCol}, nil
	case '\'':
		return token{kind: tokenQuote, lit: "'", line: startLine, col: startCol}, nil
	case '+':
		return token{kind: tokenPlus, lit: "+", line: startLine, col: startCol}, nil
	case '-':
		return token{kind: tokenMinus, lit: "-", line: startLine, col: startCol}, nil
	case '*':
		return token{kind: tokenTimes, lit: "*", line: startLine, col: startCol}, nil
	case '/':
		return token{kind: tokenDivide, lit: "/", line: startLine, col: startCol}, nil
	default:
		return token{}, fmt.Errorf("unexpected character %q at %d:%d", ch, startLine, startCol)
	}
}

func (l *lexer) skipWhitespaceAndComments() error {
	for !l.eof() {
		ch := l.peek()
		if unicode.IsSpace(ch) {
			l.advance()
			continue
		}
		if ch == '/' && l.peekN(1) == '/' {
			for !l.eof() && l.peek() != '\n' {
				l.advance()
			}
			continue
		}
		return nil
	}
	return nil
}

func (l *lexer) readIdent() string {
	start := l.pos
	for !l.eof() && isIdentPart(l.peek()) {
		l.advance()
	}
	return string(l.src[start:l.pos])
}

func (l *lexer) readInt() string {
	start := l.pos
	for !l.eof() && unicode.IsDigit(l.peek()) {
		l.advance()
	}
	return string(l.src[start:l.pos])
}

func (l *lexer) readString() (string, error) {
	startLine, startCol := l.line, l.col
	if l.peek() != '"' {
		return "", fmt.Errorf("internal string parser error at %d:%d", startLine, startCol)
	}
	l.advance() // opening quote
	start := l.pos
	for !l.eof() {
		ch := l.peek()
		if ch == '\\' {
			l.advance()
			if l.eof() {
				return "", fmt.Errorf("unterminated string at %d:%d", startLine, startCol)
			}
			l.advance()
			continue
		}
		if ch == '"' {
			lit := string(l.src[start:l.pos])
			l.advance() // closing quote
			return lit, nil
		}
		l.advance()
	}
	return "", fmt.Errorf("unterminated string at %d:%d", startLine, startCol)
}

func (l *lexer) peek() rune {
	return l.src[l.pos]
}

func (l *lexer) peekN(n int) rune {
	idx := l.pos + n
	if idx >= len(l.src) {
		return 0
	}
	return l.src[idx]
}

func (l *lexer) eof() bool {
	return l.pos >= len(l.src)
}

func (l *lexer) advance() {
	if l.eof() {
		return
	}
	if l.src[l.pos] == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	l.pos++
}

func isIdentStart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

func isIdentPart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch) || unicode.IsDigit(ch)
}
