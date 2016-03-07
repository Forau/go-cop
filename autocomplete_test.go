// Copyright (c) 2016 Forau @ github.com. MIT License.

package gocop

import (
	"testing"
)

func TestGetArgumentAutoSlice_SameNameModifySameSlice(t *testing.T) {
	s1 := getArgumentAutoSlice("test")
	s2 := getArgumentAutoSlice("test")
	sother := getArgumentAutoSlice("other")

	*s1 = append(*s1, "elem1")

	if (*s2)[0] != "elem1" {
		t.Error("Should have same pointer, but did not")
	}

	if len(*sother) > 0 {
		t.Error("We should not have affected other slice: ", sother)
	}

}

// TODO: Make it useful
func TestArgumentSugestor(t *testing.T) {
	wn := NewWorldNode()
	wn.AddSubCommand("cmd1").AddArgument("shared").AddArgument("arg2.1")
	wn.AddSubCommand("cmd2").AddArgument("shared").AddArgument("arg2.2")

	// Populate args
	sarg1 := getArgumentAutoSlice("shared")
	*sarg1 = append(*sarg1, "TheSharedArg", "Shared1")
	sarg22 := getArgumentAutoSlice("arg2.2")
	*sarg22 = append(*sarg22, "argument2.2", "other2.2")

	tokens := TokenizeRaw("cmd2 arg1 arg")
	t.Log("Tokens:", tokens)

	sug1 := wn.SugestAutoComplete(tokens)
	t.Logf("Sugestion1: %+v\n", sug1)

	if len(sug1) == 0 {
		t.Error("Did expect any sugestions yet")
	}

}
