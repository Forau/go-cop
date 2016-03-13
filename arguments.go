// Copyright (c) 2016 Forau @ github.com. MIT License.

package gocop

import (
	"log"

	"bytes"
	"strings"
)

type InvalidArgument struct {
	msg   string
	usage []string
}

func (ia *InvalidArgument) Error() string {
	var buf bytes.Buffer
	buf.WriteString(ia.msg)
	buf.WriteRune('\n')
	if len(ia.usage) > 0 {
		buf.WriteString("Close matches:\n")
		for _, u := range ia.usage {
			buf.WriteString(u)
			buf.WriteRune('\n')
		}
	}
	buf.WriteString("For full help, type: help\n")
	// This error is a bit too verbose. We should allow the reciver to decide more.
	return buf.String()
}

// Helper function to consume tokens until next command.  IE, skip whitespace after current command.
func consumeArgumentTokens(in TokenSet) (consumed, remaining TokenSet) {
	const SkipTokens = 1<<TokenWhitespace | 1<<TokenEOF
	if len(in) > 0 {
		split := 1
		for ; split < len(in) && (1<<in[split].Type&SkipTokens) != 0; split++ {
		}
		consumed = in[:split]
		if split+1 > len(in) {
			remaining = in[split:split]
		} else {
			remaining = in[split:]
		}
	}
	//	log.Printf("Splitting %+v into %+v and %+v\n", in, consumed, remaining)
	return
}

type argNodeAssignment struct {
	Node   *ArgNode
	Tokens TokenSet

	overflow TokenSet
}

// Accepts Tokens, and returns a slice of slices of the tokens not used up, and a bool to indicate acceptance
// Normally a node would return a slice with only one subslice in that starts at the token for next argument.
// However, for optional nodes, or multi-nodes, the result should be all permutations of possible accepts.
type acceptPermutationsFn func(node *ArgNode, in TokenSet) (accepted []argNodeAssignment)

func worldAcceptorFn(node *ArgNode, in TokenSet) (accepted []argNodeAssignment) {
	panic("acceptPermutationsFn should not be called on root node")
}
func commandAcceptorFn(node *ArgNode, in TokenSet) (accepted []argNodeAssignment) {
	if len(in) > 0 {
		if (len(in) == 1 && strings.Index(node.Name, in.Stringify()) == 0) ||
			(len(in) > 1 && in.Filter(TokenNoWhitespace)[0].ToString() == node.Name) {
			con, rem := consumeArgumentTokens(in)
			accepted = append(accepted, argNodeAssignment{Node: node, Tokens: con, overflow: rem})
		}
	}
	return
}
func singleArgumentAcceptorFn(node *ArgNode, in TokenSet) (accepted []argNodeAssignment) {
	con, rem := consumeArgumentTokens(in)
	if len(con) > 0 {
		accepted = append(accepted, argNodeAssignment{Node: node, Tokens: con, overflow: rem})
	}
	return
}

func repeatAcceptPerm(node *ArgNode, ap acceptPermutationsFn, prefix, in TokenSet, count int) (accepted []argNodeAssignment) {
	if count > 0 {
		for _, na := range ap(node, in) {
			fullTok := append([]Token{}, prefix...)
			fullTok = append(fullTok, na.Tokens...)
			na.Tokens = fullTok
			accepted = append(accepted, na)
			accepted = append(accepted, repeatAcceptPerm(node, ap, fullTok, na.overflow, count-1)...)
		}
	}
	return
}

type NodeTypeFlags uint64

const (
	WorldNode NodeTypeFlags = 1 << iota
	CommandNode
	ArgumentNode
	OptionalNode
	MultiArgNode
)

type ArgNode struct {
	Name      string
	Children  []*ArgNode
	TypeFlags NodeTypeFlags
	AcSugestorFn
	AcInvokerFn

	acceptPermutationsFn

	RunHandler
}

func NewWorldNode() *ArgNode {
	return &ArgNode{Name: "", AcSugestorFn: worldSugestorFn, acceptPermutationsFn: worldAcceptorFn, TypeFlags: WorldNode}
}

func (an *ArgNode) AddCustomNode(name string, acsFn AcSugestorFn, aciFn AcInvokerFn, apFn acceptPermutationsFn, typeFlags NodeTypeFlags) *ArgNode {
	n := &ArgNode{Name: name, AcSugestorFn: acsFn, AcInvokerFn: aciFn, acceptPermutationsFn: apFn, TypeFlags: typeFlags}
	for _, c := range an.Children {
		if !c.allowSibling(n) {
			log.Panicf("Sibling not allowed: %+v and %+v wont get along", n, c)
		}
	}
	an.Children = append(an.Children, n)
	return n
}

func (an *ArgNode) AddSubCommand(name string) *ArgNode {
	return an.AddCustomNode(name, commandSugestorFn, nopInvokerFn, commandAcceptorFn, CommandNode)
}

func (an *ArgNode) AddArgument(name string) *ArgNode {
	return an.AddCustomNode(name, getArgumentSugestorFn(name), getArgumentInvokerFn(name), singleArgumentAcceptorFn, ArgumentNode)
}

func (an *ArgNode) Handler(rhf RunHandlerFunc) *ArgNode {
	an.RunHandler = rhf
	return an
}

func (an *ArgNode) Weight(ts TokenSet) int {
	if an.TypeFlags&CommandNode > 0 {
		if ts.Trimmed().String() != an.Name {
			return -100 // We didn't match 100%
		} else {
			return 2
		}
	}
	return 1
}

func (an *ArgNode) Optional() *ArgNode {
	return an.Times(0, 1)
}

func (an *ArgNode) Times(min, max uint64) *ArgNode {
	if min < 0 || max < min || max < 1 {
		log.Panic("Cant deal with min ", min, " and max ", max)
	} else if (an.TypeFlags & (OptionalNode | MultiArgNode)) != 0 {
		log.Panic("Already set to optional or multi use. Can not modify again.")
	}

	if min == 0 {
		an.TypeFlags |= OptionalNode
	}
	if max > 1 {
		an.TypeFlags |= MultiArgNode
	}

	oldApFn := an.acceptPermutationsFn
	an.acceptPermutationsFn = func(node *ArgNode, in TokenSet) (accepted []argNodeAssignment) {
		oneMatchAssign := oldApFn(node, in)
		accepted = append(accepted, oneMatchAssign...)
		if min < 1 {
			accepted = append(accepted, node.assignChildNodes(in)...) // Add children, and skip self. (IE. optional)
		}
		if max > 1 {
			for _, na := range oneMatchAssign {
				// TODO: We could get duplications in the result here.  We should filter...
				accepted = append(accepted, repeatAcceptPerm(node, oldApFn, na.Tokens, na.overflow, int(max-1))...)
			}
		}
		return
	}
	return an
}

func (an *ArgNode) allowSibling(new *ArgNode) bool {
	// If we really want to be strict, we should check the whole tree
	return an.Name != new.Name // For now, just dont like to share name
}

// This method should return true if we can be optional.
// That is, if we are optional _and_ do not have children,
// or if atleast one of those children is an optional branch.
func (an *ArgNode) isOptionalBranch() bool {
	if (an.TypeFlags & OptionalNode) == OptionalNode {
		if len(an.Children) == 0 {
			return true
		}
		for _, c := range an.Children {
			if c.isOptionalBranch() {
				return true
			}
		}
	}
	return false
}

func (an *ArgNode) SugestAutoComplete(in TokenSet) []string {
	return an.Sugest(an, in)
}

func (an *ArgNode) assignChildNodes(in TokenSet) (assignments []argNodeAssignment) {
	for _, c := range an.Children {
		assignments = append(assignments, c.acceptPermutationsFn(c, in)...)
	}
	return
}

// This should generally only be called on the WorldNode. IE root node.
func (an *ArgNode) generateCommandAssingPaths(in TokenSet) (finalPaths []commandAssignPath) {
	// In the go-library they use a chan here, but why spawn go-routines when a func can do the same job?
	resultCollector := func(cap *commandAssignPath) {
		finalPaths = append(finalPaths, *cap)
	}

	for _, p := range an.assignChildNodes(in) {
		for proc := (&commandAssignPath{p}).parseNext(resultCollector); proc != nil; {
			proc = proc(resultCollector)
		}
	}

	return
}

func (an *ArgNode) Usage(prfix string) (ret []string) {
	var buf bytes.Buffer

	buf.WriteString(prfix)
	if an.TypeFlags&ArgumentNode != 0 {
		buf.WriteRune('[')
	}

	buf.WriteString(an.Name)

	if an.TypeFlags&MultiArgNode != 0 {
		buf.WriteRune('*')
	}

	if an.TypeFlags&OptionalNode != 0 {
		buf.WriteRune('?')
	}

	if an.TypeFlags&ArgumentNode != 0 {
		buf.WriteRune(']')
	}

	if len(an.Children) == 0 {
		ret = append(ret, buf.String())
	} else {
		buf.WriteRune(' ')
		pr := buf.String()
		for _, c := range an.Children {
			ret = append(ret, c.Usage(pr)...)
		}
	}
	return
}

func (an *ArgNode) InvokeCommand(input string, rc RunContext) error {
	tokens := Tokenize(input)
	if tokens.HasText() {
		paths := an.generateCommandAssingPaths(tokens)
		// log.Print("Invoked PATHS: ", paths)

		min := 0
		var path commandAssignPath

		for _, p := range paths {
			// 			log.Print(p[0].Node.Name, " = ", p.Score(), ": ", p.String())
			if p.Score() > min {
				min = p.Score()
				path = p
			}
		}

		if path != nil {
			path.Invoke(rc)
		} else {
			usage := []string{}
			cmd := strings.Split(input, " ")[0]
			for _, use := range an.Usage("\t\t") {
				if strings.Index(use, cmd) >= 0 {
					usage = append(usage, use)
				}
			}

			return &InvalidArgument{"Unknown command: " + input, usage}
		}
	}
	return nil
}

type commandAssignPath []argNodeAssignment

// Gets the last node in the path
func (cap *commandAssignPath) leaf() *argNodeAssignment {
	return &(*cap)[len(*cap)-1]
}

// Returns a array of all autocomplete options.
// The returned value must be the whole line.
func (cap *commandAssignPath) SugestAutoComplete() (ret []string) {
	leaf := cap.leaf()

	if sugestions := leaf.Node.SugestAutoComplete(leaf.Tokens); len(sugestions) > 0 {
		var prefix bytes.Buffer
		for i := 0; i < len(*cap)-1; i++ {
			prefix.WriteString((*cap)[i].Tokens.String())
		}

		for _, sug := range sugestions {
			var full bytes.Buffer
			full.Write(prefix.Bytes())
			full.WriteString(sug)
			ret = append(ret, full.String())
		}
	}
	return
}

// Nice printing for easier debugging
func (cap *commandAssignPath) String() string {
	var buffer bytes.Buffer
	for idx, nodAss := range *cap {
		if idx > 0 {
			buffer.WriteString("/")
		}

		buffer.WriteString(nodAss.Node.Name)
		buffer.WriteString("[")
		for _, t := range nodAss.Tokens {
			buffer.WriteString(t.ToString())
		}
		buffer.WriteString("]")
	}
	return buffer.String()
}

func (cap *commandAssignPath) append(ass *argNodeAssignment) {
	*cap = append(*cap, *ass)
}

func (cap *commandAssignPath) fork(ass *argNodeAssignment) *commandAssignPath {
	fork := append(commandAssignPath{}, (*cap)...)
	fork = append(fork, *ass)
	return &fork
}

// Recursive func for assigning tokens to nodes
type procAssignmentResult func(*commandAssignPath)
type procAssignmentFn func(procAssignmentResult) procAssignmentFn

func chainProcAssignments(pa ...procAssignmentFn) procAssignmentFn {
	return func(res procAssignmentResult) procAssignmentFn {
		if len(pa) == 0 {
			return nil
		}
		if next := pa[0](res); next != nil {
			return next
		}
		return chainProcAssignments(pa[1:]...)
	}
}

func (cap *commandAssignPath) parseNext(result procAssignmentResult) procAssignmentFn {
	leaf := cap.leaf()
	if len(leaf.overflow) > 0 {
		perm := leaf.Node.assignChildNodes(leaf.overflow)
		if len(perm) > 0 {
			nextCalls := []procAssignmentFn{cap.parseNext} // We will go another round.  The assignment comes after forks
			for _, fork := range perm[1:] {
				nextCalls = append(nextCalls, cap.fork(&fork).parseNext)
			}
			cap.append(&perm[0]) // We must do this after we have forked
			return chainProcAssignments(nextCalls...)
		} else if len(leaf.Node.Children) == 0 {
			// We have more then we need. This is not valid for commands or autocomplete, but we save it,
			// so we can print usage in case nothing matches
			result(cap)
		}
	} else {
		// This is full match, where we have matched all the input.
		result(cap)

		// Might still have child nodes, and if we ended with whitespace, they are candidates for autocomplete.
		// Only include them if we already have text. If not, we are the leaf for autocomplete
		if leaf.Tokens.HasText() && leaf.Tokens[len(leaf.Tokens)-1].Type == TokenEOF {
			for _, c := range leaf.Node.Children {
				newCap := append(commandAssignPath{}, (*cap)...)
				newCap = append(newCap, argNodeAssignment{Node: c})
				result(&newCap)
			}
		}
	}
	return nil
}

func (cap *commandAssignPath) Score() int {
	score := 0
	for _, ass := range *cap {
		score += ass.Node.Weight(ass.Tokens)
	}
	leaf := cap.leaf()

	if len(leaf.overflow) > 0 ||
		len(leaf.Tokens) == 0 {
		score -= 100 // We should not win
	}

	if len(leaf.Node.Children) > 0 {
		for _, c := range leaf.Node.Children {
			if c.isOptionalBranch() {
				// We have atleast one child that is totally optional to the end, so do not penalize
				return score
			}
		}
		score -= 100 // Have mandatory children only
	}

	return score
}

func (cap *commandAssignPath) Invoke(context RunContext) {
	for _, ass := range *cap {
		if ass.Node.RunHandler != nil {
			context.Handler(ass.Node.RunHandler)
		}
		ass.Node.Invoke(&ass, context)
	}
	context.Invoke()
}
