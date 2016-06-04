package gocop

import (
	"testing"
)

func assertEqual(t *testing.T, exp, act string) {
	if exp != act {
		t.Errorf("Expected '%s', but got '%s'\n", exp, act)
	}
}

func TestConsumeArgumentTokens(t *testing.T) {
	tokens := Tokenize("Start  \t\t   'Next unterminated")
	t.Log("Tokens: ", tokens)

	consumed, remaining := consumeArgumentTokens(tokens)
	t.Log("Consumed: ", consumed, ", remaining: ", remaining)

	if len(consumed) != 2 || len(remaining) != 1 {
		t.Error("Expected 3 tokens, and first + whitespaces should be consumed")
	}
}

func TestConsumeArgumentTokensWithOnlyOneToken(t *testing.T) {
	tokens := Tokenize("Start")
	t.Log("Tokens: ", tokens)

	consumed, remaining := consumeArgumentTokens(tokens)
	t.Log("Consumed: ", consumed, ", remaining: ", remaining)

	if len(consumed) != 1 || len(remaining) != 0 {
		t.Error("Expected 1 token, and it should be consumed")
	}
}

func TestNewArgNode(t *testing.T) {
	n := NewWorldNode().AddCustomNode("TEST", commandSugestorFn, nopInvokerFn, commandAcceptorFn, CommandNode)
	if n == nil {
		t.Error("Expected a object back")
	}

	tokens := Tokenize("TE")
	t.Log("Tokens: ", tokens)

	//	sugestions := n.SugestAutoComplete(tokens)
	//	t.Log("Sugestions: ", sugestions)

	//	if len(sugestions) != 1 || sugestions[0] != "TEST" {
	//		t.Error("Failed to sugest node name for autocomplete")
	//	}
}

func TestArgNode_assignChildNodes(t *testing.T) {
	n := NewWorldNode()
	c1 := n.AddCustomNode("C1", commandSugestorFn, nopInvokerFn, commandAcceptorFn, CommandNode)
	c2 := n.AddCustomNode("C2", commandSugestorFn, nopInvokerFn, commandAcceptorFn, CommandNode)
	c3 := n.AddCustomNode("C3", commandSugestorFn, nopInvokerFn, commandAcceptorFn, CommandNode)

	t.Logf("Root: %+v, C1: %+v, C1: %+v, C1: %+v\n", n, c1, c2, c3)

	tokens := Tokenize("C2 does evil stuff")
	t.Log("Tokens: ", tokens)

	assignment := n.assignChildNodes(tokens)
	t.Log("Assignment: ", assignment)

	if len(assignment) != 1 {
		t.Error("Expected only C2 to match")
		return
	}

	if len(assignment[0].overflow) != 5 {
		t.Error("Expected overflow to be three words (does evil stuff) with 2 whitespaces in between")
	}

	if assignment[0].Node != c2 {
		t.Error("Expected C2 to be selected")
	}
}

func TestArgNode_generateCommandAssingPathsWithOptionalPath(t *testing.T) {
	n := NewWorldNode()
	cmd := n.AddSubCommand("cmd")
	oarg := cmd.AddArgument("opt").Optional()
	oarg.AddArgument("noopt")
	oarg.AddSubCommand("sub")

	for _, u := range n.Usage("\t", "\t\t") {
		t.Log(u)
	}

	tokens := Tokenize("cmd sub  ")
	t.Log("Tokens: ", tokens)

	t.Logf("\nStringify: %+v\nTrimString: %+v\n", tokens.Stringify(), tokens.Trimmed().String())

	paths := n.generateCommandAssingPaths(tokens)

	for idx, p := range paths {
		t.Log("Path[", idx, "]", p.String())
	}

	if len(paths) != 5 {
		t.Error("Expected 5 paths to be returned, 3 full ones, and 2 where opt is assigned with empty children")
	}
}

func TestArgNode_SugestAutoComplete(t *testing.T) {
	n := NewWorldNode()
	arg := n.AddSubCommand("cmd").AddArgument("noopt")
	arg.AcSugestorFn = func(node *ArgNode, in TokenSet) []string {
		if in[0].ToString() == "aut" {
			return []string{"autocomplete"}
		}
		return []string{}
	}

	tokens := Tokenize("cmd aut")
	t.Log("Tokens: ", tokens)

	sugestions := n.SugestAutoComplete(tokens)
	t.Log("Sugestions: ", sugestions)

	if len(sugestions) != 1 || sugestions[0] != "cmd autocomplete" {
		t.Error("Expected 'cmd autocomplete' as sugestion")
	}
}

func TestCommandAssignPath_Invoke(t *testing.T) {
	n := NewWorldNode()
	n.AddSubCommand("cmd").AddArgument("arg1").AddArgument("arg2").Optional().AddArgument("arg3").Handler(func(rc RunContext) (interface{}, error) {
		assertEqual(t, "argument1", rc.Get("arg1"))
		assertEqual(t, "arg2\\ nr\\ 2", rc.Get("arg2"))
		assertEqual(t, "'Argument Nummer 3'", rc.Get("arg3"))
		return "Good", nil
	})
	t.Log("World node: ", n)

	tokens := Tokenize("cmd argument1 arg2\\ nr\\ 2 'Argument Nummer 3'")
	t.Log("Tokens: ", tokens)

	paths := n.generateCommandAssingPaths(tokens)
	t.Log("Paths ", paths)

	if len(paths) != 1 {
		t.Error("Expected only one path to match")
	}

	drc := &DefaultRunContext{values: make(map[string]string)}
	paths[0].Invoke(drc)

	t.Log("RunContext: ", drc)
}

func TestCommandAssignPath_GreedyPath(t *testing.T) {
	n := NewWorldNode()
	n.AddSubCommand("cmd").AddArgument("arg1").Optional().AddArgument("arg2").Times(1, 4)
	t.Log("World node: ", n)

	paths := n.generateCommandAssingPaths(Tokenize("cmd onlyArg"))
	for idx, p := range paths {
		t.Log("Path[", idx, "] ", p.Score(), " -> ", p.String())
	}

	if len(paths) != 2 && paths[0].leaf().Node.Name != paths[1].leaf().Node.Name {
		t.Error("Expected two results, where arg1 is assigned once (for autocomplete only), and arg2 the other time")
	}

	paths = n.generateCommandAssingPaths(Tokenize("cmd one two three four five six seven eight"))
	for idx, p := range paths {
		t.Log("Path[", idx, "] ", p.Score(), " -> ", p.String())
	}

	if len(paths) != 4 || paths[3].leaf().Tokens.Stringify() != "two three four five" {
		t.Log("Expected 4 results")
	}
}
