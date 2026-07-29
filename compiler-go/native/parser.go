package native

import (
	"fmt"
	"os"
	"strconv"
)

var topLevelKeywords = map[string]bool{
	"proto":   true,
	"type":    true,
	"enum":    true,
	"message": true,
	"const":   true,
	"import":  true,
	"option":  true,
}

type parser struct {
	lx  *lexer
	cur token
}

func ParseString(s string) (*Proto, error) {
	p := &parser{lx: newLexer(s)}
	if err := p.next(); err != nil {
		return nil, err
	}
	return p.parseProto()
}

func ParseFile(path string) (*Proto, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseString(string(b))
}

func (p *parser) parseProto() (*Proto, error) {
	if err := p.expectIdent("proto"); err != nil {
		return nil, err
	}
	name, err := p.expectAnyIdent()
	if err != nil {
		return nil, err
	}
	if err := p.consumeOptionalSemi(); err != nil {
		return nil, err
	}

	out := &Proto{Name: name}
	for p.cur.kind != tokenEOF {
		if p.cur.kind != tokenIdent {
			return nil, p.errf("expected top-level declaration")
		}
		switch p.cur.lit {
		case "type":
			a, err := p.parseAlias()
			if err != nil {
				return nil, err
			}
			out.Aliases = append(out.Aliases, a)
		case "enum":
			e, err := p.parseEnum()
			if err != nil {
				return nil, err
			}
			out.Enums = append(out.Enums, e)
		case "message":
			m, err := p.parseMessage()
			if err != nil {
				return nil, err
			}
			out.Messages = append(out.Messages, m)
		case "const":
			if err := p.parseConst(); err != nil {
				return nil, err
			}
		case "option":
			if err := p.parseOption(); err != nil {
				return nil, err
			}
		case "import":
			if err := p.parseImport(); err != nil {
				return nil, err
			}
		default:
			return nil, p.errf("unsupported top-level keyword %q", p.cur.lit)
		}
	}
	return out, nil
}

func (p *parser) parseAlias() (Alias, error) {
	if err := p.expectIdent("type"); err != nil {
		return Alias{}, err
	}
	name, err := p.expectAnyIdent()
	if err != nil {
		return Alias{}, err
	}
	if err := p.expect(tokenAssign); err != nil {
		return Alias{}, err
	}
	t, err := p.parseTypeExpr()
	if err != nil {
		return Alias{}, err
	}
	if err := p.consumeOptionalSemi(); err != nil {
		return Alias{}, err
	}
	return Alias{Name: name, Type: t}, nil
}

func (p *parser) parseEnum() (Enum, error) {
	if err := p.expectIdent("enum"); err != nil {
		return Enum{}, err
	}
	name, err := p.expectAnyIdent()
	if err != nil {
		return Enum{}, err
	}
	if p.cur.kind == tokenQuote {
		if err := p.expect(tokenQuote); err != nil {
			return Enum{}, err
		}
	}
	if err := p.expect(tokenColon); err != nil {
		return Enum{}, err
	}
	base, err := p.expectAnyIdent()
	if err != nil {
		return Enum{}, err
	}
	if err := p.expect(tokenLBrace); err != nil {
		return Enum{}, err
	}
	e := Enum{Name: name, Base: base}
	for p.cur.kind != tokenRBrace {
		fname, err := p.expectAnyIdent()
		if err != nil {
			return Enum{}, err
		}
		if err := p.expect(tokenAssign); err != nil {
			return Enum{}, err
		}
		val, err := p.expectInt()
		if err != nil {
			return Enum{}, err
		}
		if err := p.consumeOptionalSemi(); err != nil {
			return Enum{}, err
		}
		e.Fields = append(e.Fields, EnumField{Name: fname, Value: val})
	}
	if err := p.expect(tokenRBrace); err != nil {
		return Enum{}, err
	}
	if err := p.consumeOptionalSemi(); err != nil {
		return Enum{}, err
	}
	return e, nil
}

func (p *parser) parseMessage() (Message, error) {
	if err := p.expectIdent("message"); err != nil {
		return Message{}, err
	}
	name, err := p.expectAnyIdent()
	if err != nil {
		return Message{}, err
	}
	if p.cur.kind == tokenQuote {
		if err := p.expect(tokenQuote); err != nil {
			return Message{}, err
		}
	}
	if err := p.expect(tokenLBrace); err != nil {
		return Message{}, err
	}
	m := Message{Name: name}
	for p.cur.kind != tokenRBrace {
		if p.cur.kind == tokenIdent {
			switch p.cur.lit {
			case "message":
				nested, err := p.parseMessage()
				if err != nil {
					return Message{}, err
				}
				_ = nested
				continue
			case "enum":
				nested, err := p.parseEnum()
				if err != nil {
					return Message{}, err
				}
				_ = nested
				continue
			case "type":
				if _, err := p.parseAlias(); err != nil {
					return Message{}, err
				}
				continue
			case "const":
				if err := p.parseConst(); err != nil {
					return Message{}, err
				}
				continue
			case "option":
				if err := p.parseOption(); err != nil {
					return Message{}, err
				}
				continue
			case "import":
				if err := p.parseImport(); err != nil {
					return Message{}, err
				}
				continue
			}
		}

		t, err := p.parseTypeExpr()
		if err != nil {
			return Message{}, err
		}
		fname, err := p.expectAnyIdent()
		if err != nil {
			return Message{}, err
		}
		if err := p.expect(tokenAssign); err != nil {
			return Message{}, err
		}
		num, err := p.expectInt()
		if err != nil {
			return Message{}, err
		}
		if err := p.consumeOptionalSemi(); err != nil {
			return Message{}, err
		}
		m.Fields = append(m.Fields, MessageField{Type: t, Name: fname, Number: num})
	}
	if err := p.expect(tokenRBrace); err != nil {
		return Message{}, err
	}
	if err := p.consumeOptionalSemi(); err != nil {
		return Message{}, err
	}
	return m, nil
}

func (p *parser) parseTypeExpr() (TypeExpr, error) {
	name, err := p.expectDottedIdent()
	if err != nil {
		return TypeExpr{}, err
	}
	t := TypeExpr{Name: name}
	if p.cur.kind == tokenLBracket {
		if err := p.expect(tokenLBracket); err != nil {
			return TypeExpr{}, err
		}
		n, err := p.expectInt()
		if err != nil {
			return TypeExpr{}, err
		}
		if err := p.expect(tokenRBracket); err != nil {
			return TypeExpr{}, err
		}
		t.ArraySize = n
	}
	if p.cur.kind == tokenQuote {
		if err := p.expect(tokenQuote); err != nil {
			return TypeExpr{}, err
		}
	}
	return t, nil
}

func (p *parser) parseConst() error {
	if err := p.expectIdent("const"); err != nil {
		return err
	}
	if _, err := p.expectAnyIdent(); err != nil {
		return err
	}
	if err := p.expect(tokenAssign); err != nil {
		return err
	}
	startLine := p.cur.line
	for p.cur.kind != tokenEOF {
		if p.cur.kind == tokenSemi {
			return p.expect(tokenSemi)
		}
		if p.cur.kind == tokenIdent && topLevelKeywords[p.cur.lit] && p.cur.line > startLine {
			return nil
		}
		if err := p.next(); err != nil {
			return err
		}
	}
	return nil
}

func (p *parser) parseOption() error {
	if err := p.expectIdent("option"); err != nil {
		return err
	}
	if _, err := p.expectDottedIdent(); err != nil {
		return err
	}
	if err := p.expect(tokenAssign); err != nil {
		return err
	}
	if err := p.consumeValueToken(); err != nil {
		return err
	}
	return p.consumeOptionalSemi()
}

func (p *parser) parseImport() error {
	if err := p.expectIdent("import"); err != nil {
		return err
	}
	if p.cur.kind == tokenIdent {
		if _, err := p.expectAnyIdent(); err != nil {
			return err
		}
	}
	if p.cur.kind != tokenString {
		return p.errf("expected import path string")
	}
	if err := p.next(); err != nil {
		return err
	}
	return p.consumeOptionalSemi()
}

func (p *parser) next() error {
	t, err := p.lx.nextToken()
	if err != nil {
		return err
	}
	p.cur = t
	return nil
}

func (p *parser) expect(k tokenKind) error {
	if p.cur.kind != k {
		return p.errf("unexpected token %q", p.cur.lit)
	}
	return p.next()
}

func (p *parser) expectIdent(lit string) error {
	if p.cur.kind != tokenIdent || p.cur.lit != lit {
		return p.errf("expected %q", lit)
	}
	return p.next()
}

func (p *parser) expectAnyIdent() (string, error) {
	if p.cur.kind != tokenIdent {
		return "", p.errf("expected identifier")
	}
	v := p.cur.lit
	return v, p.next()
}

func (p *parser) expectDottedIdent() (string, error) {
	base, err := p.expectAnyIdent()
	if err != nil {
		return "", err
	}
	parts := []string{base}
	for p.cur.kind == tokenDot {
		if err := p.expect(tokenDot); err != nil {
			return "", err
		}
		n, err := p.expectAnyIdent()
		if err != nil {
			return "", err
		}
		parts = append(parts, n)
	}
	return joinDot(parts), nil
}

func (p *parser) expectInt() (int, error) {
	if p.cur.kind != tokenInt {
		return 0, p.errf("expected integer")
	}
	v, err := strconv.Atoi(p.cur.lit)
	if err != nil {
		return 0, p.errf("invalid integer %q", p.cur.lit)
	}
	return v, p.next()
}

func (p *parser) consumeOptionalSemi() error {
	if p.cur.kind == tokenSemi {
		return p.expect(tokenSemi)
	}
	return nil
}

func (p *parser) consumeValueToken() error {
	switch p.cur.kind {
	case tokenString, tokenInt, tokenIdent:
		return p.next()
	default:
		return p.errf("expected option value")
	}
}

func joinDot(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "." + parts[i]
	}
	return out
}

func (p *parser) errf(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return fmt.Errorf("parse error at %d:%d: %s", p.cur.line, p.cur.col, msg)
}
