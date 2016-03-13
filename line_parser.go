// Copyright (c) 2016 Forau @ github.com. MIT License.

package gocop

import (
	"unicode"
	"unicode/utf8"

	"strings"
)

// Scanned token
type Token struct {
	Type       TokenType // Type
	val        string    // Value
	incomplete bool      // If it got terminated by eol
}

// Returns the string-value. If it is a quoted string, then omit the quotes
func (t *Token) ToString() string {
	start := 0
	end := len(t.val)
	switch t.Type {
	case TokenDQuoted, TokenSQuoted:
		start = 1
		if !t.incomplete {
			end -= 1
		}
	}
	return t.val[start:end]
}

func (t *Token) IsWhitespace() bool {
	return t.Type&TokenAllWhitespace != 0
}

// Will use -1 as EOF rune
const eof = -1

type TokenType uint64

const (
	TokenError TokenType = 1 << iota
	TokenEOF
	TokenString
	TokenDQuoted
	TokenSQuoted
	TokenWhitespace

	TokenNoWhitespace  = TokenString | TokenDQuoted | TokenSQuoted
	TokenAllWhitespace = TokenEOF | TokenWhitespace
)

type TokenSet []Token

func (ts TokenSet) String() string {
	ret := []byte{}
	for _, t := range ts {
		ret = append(ret, []byte(t.val)...)
	}
	return string(ret)
}

// Returns the string, striped of head and trail whitespaces.
func (ts TokenSet) Stringify() string {
	return ts.Trimmed().String()
}

// Returns a slice without leading or trailing whitespaces
func (ts TokenSet) Trimmed() TokenSet {
	foundNoneWS := false
	start := 0
	end := 0

	for idx, t := range ts {
		if !t.IsWhitespace() {
			end = idx + 1
			if !foundNoneWS {
				foundNoneWS = true
				start = idx
			}
		}
	}
	return ts[start:end]
}

func (ts TokenSet) Filter(keep TokenType) TokenSet {
	b := []Token{}
	for _, t := range ts {
		if (t.Type & keep) != 0 {
			b = append(b, t)
		}
	}
	return b
}

// Checks if the set contains printable characters
func (ts TokenSet) HasText() bool {
	for _, t := range ts {
		if !t.IsWhitespace() {
			return true
		}
	}
	return false
}

func (ts TokenSet) StartsWithIgnoreCase(cmp string) bool {
	lval := strings.ToLower(ts.Trimmed().String())
	return strings.Index(lval, strings.ToLower(cmp)) == 0
}

// stateFn returns a new stateFn or nil.
type stateFn func(*scanner) stateFn
type acceptFn func(r rune) bool

// scanner object.
// This is built much like https://golang.org/src/text/template/parse/lex.go but
// a bit simpler. It is probably overkill for our simple command parsing requirements,
// but the spliters and parsers in the normal packages didn't give enough flexability
// to scan, parse backslashes and return incomplete tokens, without writing some ugly for-loops.
// Also, a small lex'er is more fun.
type scanner struct {
	input string

	state stateFn // Current state

	start int
	pos   int

	tokens chan Token // Where to emit the result
}

func (s *scanner) next() rune {
	if s.pos >= len(s.input) {
		return eof
	}
	r, w := utf8.DecodeRuneInString(s.input[s.pos:])
	s.pos += w
	return r
}

func (s *scanner) peek() rune {
	backup := s.pos
	r := s.next()
	s.pos = backup
	return r
}

func (s *scanner) skip() {
	if s.start < s.pos {
		s.emit(TokenWhitespace, false)
	}
}

func (s *scanner) emit(t TokenType, incomp bool) {
	s.tokens <- Token{t, s.input[s.start:s.pos], incomp}
	s.start = s.pos
}

func (s *scanner) acceptWhile(af acceptFn) {
	backup := s.pos
	for r := s.next(); r != eof && af(r); r = s.next() {
		backup = s.pos
	}
	s.pos = backup
}

func (s *scanner) run() {
	for s.state != nil {
		s.state = s.state(s)
	}
	close(s.tokens)
}

func Tokenize(input string) (tokens TokenSet) {
	s := scanner{
		input:  input,
		state:  scanStart,
		tokens: make(chan Token, 2),
	}
	go s.run()
	for tok := range s.tokens {
		tokens = append(tokens, tok)
	}
	return
}

//
func buildEscapeSafeAcceptFn(af acceptFn) acceptFn {
	skipOne := false
	return func(r rune) bool {
		if skipOne {
			skipOne = false
			return true
		} else if r == '\\' {
			skipOne = true
			return true
		}
		return af(r)
	}
}

// States
// scanStart is the defaultState, which will search for the start of another state and switch to that
func scanStart(s *scanner) stateFn {
	s.acceptWhile(unicode.IsSpace)
	switch s.peek() {
	case eof:
		s.emit(TokenEOF, false)
		break
	case '"':
		s.skip()
		return makeGenericTypeScanner(TokenDQuoted, true, scanStart,
			accepTimes(1), untilRuneAcceptFn('"'), accepTimes(1))
	case '\'':
		s.skip()
		return makeGenericTypeScanner(TokenSQuoted, true, scanStart,
			accepTimes(1), untilRuneAcceptFn('\''), accepTimes(1))
	default:
		s.skip()
		return makeGenericTypeScanner(TokenString, true, scanStart, invertAcceptFn(unicode.IsSpace))
	}
	return nil
}

// Helper acceptFn.  Accepts N characters, no questions asked
func accepTimes(n int) acceptFn {
	return func(r rune) bool {
		n -= 1
		return n >= 0
	}
}

// Run first function until false, then next one until all are gone, then we return false
func chainAcceptFn(anArr ...acceptFn) (acceptFn, *int) {
	var methLeft int
	return func(r rune) bool {
		for {
			methLeft = len(anArr)
			if len(anArr) == 0 {
				return false
			}
			if !anArr[0](r) {
				anArr = anArr[1:]
			} else {
				methLeft--
				return true
			}
		}
	}, &methLeft
}

// Accepts until a rune is found
func untilRuneAcceptFn(rarr ...rune) acceptFn {
	return func(r rune) bool {
		for _, r0 := range rarr {
			if r0 == r {
				return false // Break accept
			}
		}
		return true
	}
}

// Inverts an accept func
func invertAcceptFn(af acceptFn) acceptFn {
	return func(r rune) bool {
		return !af(r)
	}
}

// Creates a generic scanner. Will scan for one type, and then return to scanStart state
func makeGenericTypeScanner(typ TokenType, backslashSafe bool, nextState stateFn, afs ...acceptFn) stateFn {
	af, methLeft := chainAcceptFn(afs...)
	return func(s *scanner) stateFn {
		if backslashSafe {
			s.acceptWhile(buildEscapeSafeAcceptFn(af))
		} else {
			s.acceptWhile(af)
		}

		if eof == s.peek() {
			s.emit(typ, *methLeft > 0) // If we have acceptFn that have not run, set incomplete
			return nil
		}
		s.emit(typ, false)
		return nextState
	}
}
