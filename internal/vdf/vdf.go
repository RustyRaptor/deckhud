// Package vdf parses Valve's text KeyValues format (".vdf"), as used by Steam
// Input controller configurations.
//
// Keys may repeat (a group has many "binding" entries), so a block keeps its
// children as an ordered list rather than a map. Key lookups ignore case, as
// Steam does. All accessors are safe to call on a nil *Node, which makes deep
// optional lookups such as n.Get("a").Get("b").Str("c") cheap to write.
package vdf

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Node is either a leaf (Key/Value) or a block (Key/Children).
type Node struct {
	Key      string
	Value    string
	Children []*Node // nil for leaves, non-nil (possibly empty) for blocks
	Line     int     // 1-based line where the key appeared
}

// IsBlock reports whether n is a block rather than a leaf.
func (n *Node) IsBlock() bool { return n != nil && n.Children != nil }

// Get returns the first child whose key matches (case-insensitively), or nil.
func (n *Node) Get(key string) *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if strings.EqualFold(c.Key, key) {
			return c
		}
	}
	return nil
}

// All returns every child whose key matches, in file order.
func (n *Node) All(key string) []*Node {
	if n == nil {
		return nil
	}
	var out []*Node
	for _, c := range n.Children {
		if strings.EqualFold(c.Key, key) {
			out = append(out, c)
		}
	}
	return out
}

// Str returns the value of the first leaf child with the given key, or "".
func (n *Node) Str(key string) string {
	if c := n.Get(key); c != nil && !c.IsBlock() {
		return c.Value
	}
	return ""
}

// Parse reads a whole document. The returned root is a synthetic block whose
// children are the document's top-level entries.
func Parse(r io.Reader) (*Node, error) {
	br := bufio.NewReader(r)
	if c, _, err := br.ReadRune(); err == nil && c != 0xFEFF {
		_ = br.UnreadRune() // Valve's stock templates start with a UTF-8 BOM
	}
	p := &parser{lx: lexer{r: br, line: 1}}
	root := &Node{Children: []*Node{}}
	if err := p.parseBlock(root, true); err != nil {
		return nil, err
	}
	return root, nil
}

type tokKind int

const (
	tokEOF tokKind = iota
	tokString
	tokOpen
	tokClose
	tokCond // [$PLATFORM] conditional, ignored
)

type token struct {
	kind tokKind
	text string
	line int
}

type parser struct {
	lx     lexer
	peeked *token
}

func (p *parser) next() (token, error) {
	if p.peeked != nil {
		t := *p.peeked
		p.peeked = nil
		return t, nil
	}
	return p.lx.next()
}

func (p *parser) peek() (token, error) {
	if p.peeked == nil {
		t, err := p.lx.next()
		if err != nil {
			return t, err
		}
		p.peeked = &t
	}
	return *p.peeked, nil
}

func (p *parser) parseBlock(parent *Node, top bool) error {
	for {
		t, err := p.next()
		if err != nil {
			return err
		}
		switch t.kind {
		case tokEOF:
			if !top {
				return fmt.Errorf("vdf: line %d: unterminated block %q", parent.Line, parent.Key)
			}
			return nil
		case tokClose:
			if top {
				return fmt.Errorf("vdf: line %d: unexpected '}'", t.line)
			}
			return nil
		case tokOpen, tokCond:
			return fmt.Errorf("vdf: line %d: expected key", t.line)
		}

		node := &Node{Key: t.text, Line: t.line}
		v, err := p.next()
		if err != nil {
			return err
		}
		if v.kind == tokCond {
			if v, err = p.next(); err != nil {
				return err
			}
		}
		switch v.kind {
		case tokString:
			node.Value = v.text
			if c, err := p.peek(); err != nil {
				return err
			} else if c.kind == tokCond {
				p.peeked = nil
			}
		case tokOpen:
			node.Children = []*Node{}
			if err := p.parseBlock(node, false); err != nil {
				return err
			}
		default:
			return fmt.Errorf("vdf: line %d: key %q has no value", t.line, t.text)
		}
		parent.Children = append(parent.Children, node)
	}
}

type lexer struct {
	r    *bufio.Reader
	line int
}

func (l *lexer) read() (rune, bool) {
	c, _, err := l.r.ReadRune()
	if err != nil {
		return 0, false
	}
	if c == '\n' {
		l.line++
	}
	return c, true
}

func (l *lexer) unread(c rune) {
	_ = l.r.UnreadRune()
	if c == '\n' {
		l.line--
	}
}

func (l *lexer) next() (token, error) {
	for {
		c, ok := l.read()
		if !ok {
			return token{kind: tokEOF, line: l.line}, nil
		}
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			continue
		case c == '/':
			if n, ok := l.read(); ok && n == '/' {
				for {
					c, ok := l.read()
					if !ok || c == '\n' {
						break
					}
				}
				continue
			} else if ok {
				l.unread(n)
			}
			return l.bare(c)
		case c == '{':
			return token{kind: tokOpen, line: l.line}, nil
		case c == '}':
			return token{kind: tokClose, line: l.line}, nil
		case c == '[':
			start := l.line
			for {
				c, ok := l.read()
				if !ok {
					return token{}, fmt.Errorf("vdf: line %d: unterminated conditional", start)
				}
				if c == ']' {
					return token{kind: tokCond, line: start}, nil
				}
			}
		case c == '"':
			return l.quoted()
		default:
			return l.bare(c)
		}
	}
}

func (l *lexer) quoted() (token, error) {
	start := l.line
	var b strings.Builder
	for {
		c, ok := l.read()
		if !ok {
			return token{}, fmt.Errorf("vdf: line %d: unterminated string", start)
		}
		switch c {
		case '"':
			return token{kind: tokString, text: b.String(), line: start}, nil
		case '\\':
			n, ok := l.read()
			if !ok {
				return token{}, fmt.Errorf("vdf: line %d: unterminated string", start)
			}
			switch n {
			case 'n':
				b.WriteRune('\n')
			case 't':
				b.WriteRune('\t')
			case '\\', '"':
				b.WriteRune(n)
			default:
				b.WriteRune('\\')
				b.WriteRune(n)
			}
		default:
			b.WriteRune(c)
		}
	}
}

func (l *lexer) bare(first rune) (token, error) {
	start := l.line
	var b strings.Builder
	b.WriteRune(first)
	for {
		c, ok := l.read()
		if !ok {
			break
		}
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '"' || c == '{' || c == '}' || c == '[' {
			l.unread(c)
			break
		}
		b.WriteRune(c)
	}
	return token{kind: tokString, text: b.String(), line: start}, nil
}
