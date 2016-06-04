package gocop

import (
	"log"
	"testing"
)

var _ testing.T

func ExampleCommandParser_NewWorld() {
	cp := NewCommandParser()

	// New world. IE, root node
	world := cp.NewWorld()
	// Add command1 with mandatory argument
	world.AddSubCommand("command1").AddArgument("argument")
	// Add command2 with optional argument
	world.AddSubCommand("command2").AddArgument("argument").Optional()

	// Print usage
	for _, u := range world.Usage("\t", "\t\t") {
		log.Print(u)
	}
}
