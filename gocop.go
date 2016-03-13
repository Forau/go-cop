// Copyright (c) 2016 Forau @ github.com. MIT License.

// GO-COP is a utility to add functions easy to use from the tty.
// Autocomplete sugestions is context aware, and the command structure is quite liberal,
// meening that multiple optional or greedy arguments are allowed, though not encurraged.
package gocop

import (
	"errors"
	"fmt"
	"log"

	"github.com/peterh/liner"
)

type RunHandlerFunc func(rc RunContext)

func (rh RunHandlerFunc) HandleCommand(rc RunContext) {
	rh(rc)
}

type SugestionProvider struct {
}

type RunHandler interface {
	HandleCommand(rc RunContext)
}

type RunContext interface {
	Put(name, value string)
	Get(name string) string

	SugestionProvider() SugestionProvider

	Handler(rh RunHandler)
	Invoke()
}

type RunContextProviderFn func(cp *CommandParser) RunContext

func DefaultRunContextProvider(cp *CommandParser) RunContext {
	return &DefaultRunContext{
		values:            make(map[string]string),
		sugestionProvider: cp.SugestionProvider,
	}
}

type DefaultRunContext struct {
	values            map[string]string
	handler           RunHandler
	sugestionProvider SugestionProvider
}

func (drc *DefaultRunContext) Put(name, value string) {
	drc.values[name] = value
}
func (drc *DefaultRunContext) Get(name string) string {
	return drc.values[name]
}
func (drc *DefaultRunContext) Handler(h RunHandler) {
	drc.handler = h
}
func (drc *DefaultRunContext) SugestionProvider() SugestionProvider {
	return drc.sugestionProvider
}

func (drc *DefaultRunContext) Invoke() {
	if drc.handler != nil {
		drc.handler.HandleCommand(drc)
		// log.Print("Invoked with values: ", drc.values)
	} else {
		log.Printf("Executing RunContext %+v, but did not get a run function...\n", drc)
	}
}

type CommandParser struct {
	liner *liner.State

	world *ArgNode

	rcProvider        RunContextProviderFn
	SugestionProvider SugestionProvider
}

func NewCommandParser() *CommandParser {
	return &CommandParser{
		rcProvider:        DefaultRunContextProvider,
		SugestionProvider: SugestionProvider{},
	}
}

func (cp *CommandParser) AutoCompleter(line string) (c []string) {
	if cp.world != nil {
		tokens := Tokenize(line)
		c = append(c, cp.world.SugestAutoComplete(tokens)...)
	}
	return // TODO: Implement
}

func (cp *CommandParser) NewWorld() *ArgNode {
	cp.world = NewWorldNode()

	// Add standard commands. This might be optional later
	cp.AddStandardCommands(cp.world)

	return cp.world
}

func (cp *CommandParser) AddStandardCommands(an *ArgNode) {
	an.AddSubCommand("help").Handler(cp.printHelp).AddArgument("help_argument").Optional()
}

func (cp *CommandParser) printHelp(rc RunContext) {
	fmt.Println("Usage:")
	for _, u := range cp.world.Usage("\t\t") {
		fmt.Println(u)
	}
}

func (cp *CommandParser) NewRunContext() RunContext {
	return cp.rcProvider(cp)
}

func (cp *CommandParser) MainLoop() (err error) {
	defer func() {
		switch x := recover().(type) {
		case string:
			err = errors.New(x)
		case error:
			err = x
		default:
			err = errors.New("Unknown panic")
		}
	}()

	cp.liner = liner.NewLiner()
	defer cp.liner.Close()
	cp.liner.SetCtrlCAborts(true)
	cp.liner.SetCompleter(cp.AutoCompleter)

	for {
		fmt.Print("\x1b[0;33m")
		l, err := cp.liner.Prompt("➜ ")
		if err != nil {
			panic(err)
		}
		cp.liner.AppendHistory(l)
		fmt.Printf("\x1b[0;36m")
		err = cp.world.InvokeCommand(l, cp.NewRunContext())
		fmt.Print("\x1b[0m")
		if err != nil {
			fmt.Print(err)
		}
	}
}
