package dsl

import (
	"fmt"
	"strings"
)

func ParsePipeline(input string) (Pipeline, error) {
	tokens, err := lex(input)
	if err != nil {
		return Pipeline{}, err
	}
	if len(tokens) == 0 {
		return Pipeline{}, newProtocolError("invalid_syntax", "empty input")
	}

	var p Pipeline
	var current *Command

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		switch tok.kind {
		case tokenWord:
			if current == nil {
				p.Commands = append(p.Commands, Command{Name: tok.value})
				current = &p.Commands[len(p.Commands)-1]
				continue
			}
			current.Args = append(current.Args, tok.value)
		case tokenPipe:
			if current == nil {
				return Pipeline{}, newProtocolError("invalid_syntax", "pipe without command")
			}
			current = nil
		case tokenRedirect:
			if current == nil {
				return Pipeline{}, newProtocolError("invalid_syntax", "redirect without command")
			}
			if i+1 >= len(tokens) {
				return Pipeline{}, newProtocolError("missing_argument", "redirect path is required")
			}
			next := tokens[i+1]
			if next.kind != tokenWord {
				return Pipeline{}, newProtocolError("invalid_syntax", "redirect target must be a path")
			}
			if p.Redirect != nil {
				return Pipeline{}, newProtocolError("unsupported_syntax", "multiple redirects are not supported")
			}
			p.Redirect = &Redirect{Path: next.value}
			if i+2 != len(tokens) {
				return Pipeline{}, newProtocolError("unsupported_syntax", "tokens after redirect are not supported")
			}
			i++
		default:
			return Pipeline{}, newProtocolError("unsupported_syntax", fmt.Sprintf("unsupported token %q", tok.value))
		}
	}

	if current == nil {
		return Pipeline{}, newProtocolError("invalid_syntax", "pipeline cannot end with a pipe")
	}
	if len(p.Commands) == 0 {
		return Pipeline{}, newProtocolError("invalid_syntax", "no commands found")
	}

	return p, nil
}

type tokenKind int

const (
	tokenWord tokenKind = iota
	tokenPipe
	tokenRedirect
)

type token struct {
	kind  tokenKind
	value string
}

func lex(input string) ([]token, error) {
	var tokens []token
	var buf strings.Builder
	var quote rune

	flush := func() {
		if buf.Len() == 0 {
			return
		}
		tokens = append(tokens, token{kind: tokenWord, value: buf.String()})
		buf.Reset()
	}

	for _, r := range input {
		if quote != 0 {
			if r == quote {
				quote = 0
				continue
			}
			buf.WriteRune(r)
			continue
		}

		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t', '\n', '\r':
			flush()
		case '|':
			flush()
			tokens = append(tokens, token{kind: tokenPipe, value: "|"})
		case '>':
			flush()
			tokens = append(tokens, token{kind: tokenRedirect, value: ">"})
		case ';', '&', '<', '`', '$':
			return nil, newProtocolError("unsupported_syntax", fmt.Sprintf("unsupported character %q", string(r)))
		case '*', '?', '[', ']':
			return nil, newProtocolError("unsupported_syntax", fmt.Sprintf("globbing is not supported: %q", string(r)))
		default:
			buf.WriteRune(r)
		}
	}

	if quote != 0 {
		return nil, newProtocolError("invalid_syntax", "unterminated quoted string")
	}

	flush()
	return tokens, nil
}
