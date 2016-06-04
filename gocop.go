// Copyright (c) 2016 Forau @ github.com. MIT License.

// GO-COP is a utility to add functions easy to use from the tty.
// Autocomplete sugestions is context aware, and the command structure is quite liberal,
// meening that multiple optional or greedy arguments are allowed, though not encurraged.
package gocop

import (
	"errors"
	"fmt"

	"github.com/peterh/liner"
)

type RunHandlerFunc func(rc RunContext) (interface{}, error)

func (rh RunHandlerFunc) HandleCommand(rc RunContext) (interface{}, error) {
	return rh(rc)
}

type SugestionProvider struct {
}

type RunHandler interface {
	HandleCommand(rc RunContext) (interface{}, error)
}

type ResultHandlerFn func(interface{}, error)

var DefaultResultHandler = func(in interface{}, err error) {
	if err != nil {
		fmt.Print("\x1b[0;31m")
		fmt.Print(err)
	} else {
		fmt.Print("\x1b[0;35m")
		fmt.Printf("%+v\n", in)
	}
}

type RunContext interface {
	Put(name, value string)
	Get(name string) string

	SugestionProvider() SugestionProvider

	Handler(rh RunHandler)
	Invoke() (interface{}, error)
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

func (drc *DefaultRunContext) Invoke() (interface{}, error) {
	if drc.handler != nil {
		return drc.handler.HandleCommand(drc)
	} else {
		return nil, fmt.Errorf("Executing RunContext %+v, but did not get a run function...\n", drc)
	}
}

type CommandParser struct {
	liner *liner.State

	world *ArgNode

	rcProvider        RunContextProviderFn
	SugestionProvider SugestionProvider
	ResultHandler     ResultHandlerFn
}

func NewCommandParser() *CommandParser {
	return &CommandParser{
		rcProvider:        DefaultRunContextProvider,
		SugestionProvider: SugestionProvider{},
		ResultHandler:     DefaultResultHandler,
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

func (cp *CommandParser) printHelp(rc RunContext) (interface{}, error) {
	fmt.Println("Usage:")
	for _, u := range cp.world.Usage("\t\t", "\t\t\t") {
		fmt.Println(u)
	}
	return nil, nil
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
		res, err := cp.world.InvokeCommand(l, cp.NewRunContext())
		cp.ResultHandler(res, err)
		fmt.Print("\x1b[0m")
	}
}
