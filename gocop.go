// Copyright (c) 2016 Forau @ github.com. MIT License.

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

type RunHandler interface {
	HandleCommand(rc RunContext)
}

type RunContext interface {
	SetValue(name, value string)
	GetValue(name string) string

	SetHandler(rh RunHandler)
	Invoke()
}

type DefaultRunContext struct {
	values  map[string]string
	handler RunHandler
}

func (drc *DefaultRunContext) SetValue(name, value string) {
	if drc.values == nil {
		drc.values = make(map[string]string)
	}
	drc.values[name] = value
}
func (drc *DefaultRunContext) GetValue(name string) string {
	return drc.values[name]
}
func (drc *DefaultRunContext) SetHandler(h RunHandler) {
	drc.handler = h
}
func (drc *DefaultRunContext) Invoke() {
	if drc.handler != nil {
		drc.handler.HandleCommand(drc)
		log.Print("Invoked with values: ", drc.values)
	} else {
		log.Printf("Executing RunContext %+v, but did not get a run function...\n", drc)
	}
}

type CommandParser struct {
	liner *liner.State

	world *ArgNode
}

func NewCommandParser() *CommandParser {
	return &CommandParser{}
}

func (cp *CommandParser) AutoCompleter(line string) (c []string) {
	if cp.world != nil {
		tokens := TokenizeRaw(line)
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
		err = cp.world.InvokeCommand(l, &DefaultRunContext{})
		fmt.Print("\x1b[0m")
		if err != nil {
			fmt.Print(err)
		}
	}
	return
}
