package gocop

import (
	"strings"
	"testing"

	"log"
)

func TestTokenSet_Trimmed(t *testing.T) {
	trimAndCompare := func(input string, trimmedSizeDiff int) {
		tok := Tokenize(input)
		tokTrim := tok.Trimmed()
		t.Logf("IN: '%s' -> '%+v' Trimmed()-> '%+v'\n", input, tok, tokTrim)
		if len(tok) != trimmedSizeDiff+len(tokTrim) {
			t.Error("Expected size diff to be ", trimmedSizeDiff, " but got ", len(tok), " vs ", len(tokTrim))
		}
	}

	trimAndCompare("This should \nNOT\t be trimmed", 0)
	trimAndCompare("  This should only trim the head", 1)
	trimAndCompare("This should only trim the tail   ", 1)
	trimAndCompare("    This should only trim both   ", 2)
	trimAndCompare("", 1) // This seems weird, but "" generates a EOF-token, and it will be trimmed

	if len(TokenSet{}.Trimmed()) != 0 {
		t.Error("Should have been able to trim a empty token set")
	}
}

func TestAcceptWhile_simple(t *testing.T) {
	testStr := "Detta ar test string"
	scanner := scanner{input: testStr}

	scanner.acceptWhile(func(r rune) bool {
		return r != ' '
	})

	if scanner.pos != strings.Index(testStr, " ") {
		t.Error("Expected same index as strings.Index: ", scanner.pos, " != ", strings.Index(testStr, " "))
	}
}

func TestAcceptWhile_EscapeWrapper(t *testing.T) {
	testStr := "Skip this \\\" quote. \" is the one"
	scanner := scanner{input: testStr}

	escScan := buildEscapeSafeAcceptFn(func(r rune) bool {
		return r != '"'
	})

	scanner.acceptWhile(escScan)

	if scanner.pos != strings.LastIndex(testStr, "\"") {
		t.Error("Expected same index as strings.Index: ", scanner.pos, " != ", strings.Index(testStr, " "))
	}
}

func TestScanCommandString_BasicStart(t *testing.T) {
	testStr := "     \t\n Start with\\ some \n\n\t whitespaces"
	expected := []string{"Start", "with\\ some", "whitespaces"}
	tokens := Tokenize(testStr).Filter(TokenNoWhitespace)
	t.Log("Got tokens:", tokens)

	for i := 0; i < len(expected); i++ {
		if tokens[i].val != expected[i] {
			t.Error("Expected '", expected[i], "' but got '", tokens[i].val, "'")
		}
	}
}

func TestTokenizeDifferentTypes(t *testing.T) {
	testStr := `
Simple "Double \" Quoted" 
And \'  'Single\' Quoted'  `

	expected := []string{"Simple", "\"Double \\\" Quoted\"", "And",
		"\\'", "'Single\\' Quoted'"}

	t.Log("Tokenizing ", testStr)
	tokens := Tokenize(testStr).Filter(TokenNoWhitespace)
	t.Log("Got tokens:", tokens)

	for i := 0; i < len(expected); i++ {
		if tokens[i].val != expected[i] {
			t.Error("Expected '", expected[i], "' but got '", tokens[i], "'")
		}
	}
}

func TestTokenizeNoneTerminatedDQString(t *testing.T) {
	testStr := "\"And not terminate"
	tokens := Tokenize(testStr)
	t.Log("Got tokens:", tokens)

	tok := tokens[0]
	if !tok.incomplete || tok.val != testStr {
		t.Error("Expected one token with unterminated string, but got ", tokens)
	}
	if tok.ToString() != "And not terminate" {
		t.Error("Failed to remove quote in ToString method")
	}
}

func ExampleTokenSet_Filter() {
	tokens := Tokenize("\"A \"'input' string")

	log.Print("Initial tokens: ", tokens)

	log.Print("Filtered on not whitespace: ", tokens.Filter(TokenNoWhitespace))

	log.Print("Filtered on single or double quoted string: ", tokens.Filter(TokenSQuoted|TokenDQuoted))
}
