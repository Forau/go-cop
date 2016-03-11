// Copyright (c) 2016 Forau @ github.com. MIT License.

package gocop

import (
	"strings"
)

var (
	argumentAutoMap = make(map[string]*[]string)
)

func getArgumentAutoSlice(name string) *[]string {
	slice := argumentAutoMap[name]
	if slice == nil {
		slice = &[]string{}
		argumentAutoMap[name] = slice
	}
	return slice
}

// A function to customize autocomplete replies.
// Uses a slice of Token as input to make it possible for multi-match types. IE, match all remaining commands
type AcSugestorFn func(node *ArgNode, in TokenSet) []string

func (asf AcSugestorFn) Sugest(node *ArgNode, in TokenSet) []string {
	return asf(node, in)
}

type AcSugestor interface {
	Sugest(node *ArgNode, in TokenSet) []string
}

func worldSugestorFn(node *ArgNode, in TokenSet) (res []string) {
	paths := node.generateCommandAssingPaths(in)
	for _, p := range paths {
		sac := p.SugestAutoComplete()
		res = append(res, sac...)
	}
	return
}

// AcSugestorFn for commands
func commandSugestorFn(node *ArgNode, in TokenSet) (ret []string) {
	if len(in) == 1 && strings.Index(node.Name, in[0].val) == 0 {
		return []string{node.Name}
	}
	return
}

func getArgumentSugestorFn(name string) AcSugestorFn {
	// Since the AcSugestorFn can run concurrently, we do the map-lookup during init
	sugestionSlice := getArgumentAutoSlice(name)
	return func(node *ArgNode, in TokenSet) (ret []string) {
		if len(in) <= 1 {
			val := in.String()
			if len(val) > 0 {
				for _, ss := range *sugestionSlice {
					if strings.Index(ss, val) == 0 {
						ret = append(ret, ss)
					}
				}
			} else {
				ret = append(ret, *sugestionSlice...)
			}
		}
		return
	}
}

// A function that registers and applies the value on resultMap
type AcInvokerFn func(assignment *argNodeAssignment, context RunContext)

func (aif AcInvokerFn) Invoke(assignment *argNodeAssignment, context RunContext) {
	aif(assignment, context)
}

type AcInvoker interface {
	Invoke(assignment *argNodeAssignment, context RunContext)
}

func nopInvokerFn(assignment *argNodeAssignment, context RunContext) {
}

func getArgumentInvokerFn(name string) AcInvokerFn {
	sugestionSlice := getArgumentAutoSlice(name)
	return func(assignment *argNodeAssignment, context RunContext) {
		val := assignment.Tokens.Stringify()
		*sugestionSlice = append(*sugestionSlice, val)
		context.Put(name, val)
	}
}
