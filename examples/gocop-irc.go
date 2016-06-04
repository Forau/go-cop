// Copyright (c) 2016 Forau @ github.com. MIT License.

// This is just an example. See it as a gist or something, not as a full irc-client.
// Also, it is a work in progress to test out gocop features

package main

import (
	"github.com/Forau/gocop"
	"net"

	"log"
	"os"
	"time"

	"bufio"

	"fmt"
	"strings"
)

var (
	skipUserModeRepl = strings.NewReplacer("@", "", "+", "")
)

type IrcConn struct {
	nick string // Our client is simple, so will be used as user too

	socket net.Conn      // Reference to the connection
	end    chan struct{} // Channel to close down.

	write chan string

	listeners []func(ic *IrcConn, evt *IrcEvent)
}

type IrcEvent struct {
	Raw     string
	Tags    string
	Prefix  string
	Command string
	Params  string
}

func splitTo(in []byte, sep byte) (pre, rest []byte) {
	i := 0
	for ; i < len(in); i++ {
		if in[i] == sep {
			i += 1 // We know the separator, so lets skip it
			break
		}
		pre = append(pre, in[i])
	}
	rest = append(rest, in[i:]...)
	return
}

func (ic *IrcConn) Init() *IrcConn {

	// Add output
	ic.AddListener(func(ic *IrcConn, evt *IrcEvent) {
		log.Print(evt.Raw)
	})

	// Add ping reply
	ic.AddListener(func(ic *IrcConn, evt *IrcEvent) {
		if evt.Command == "PING" {
			ic.SendRaw("PONG " + evt.Params)
		}
	})

	ic.AddListener(func(ic *IrcConn, evt *IrcEvent) {
		switch evt.Command {
		case "353": // Channel names
			// :b0rk.uk.quakenet.org 353 Trizzla = #wow-revive :Trizzla @Munorion @Q dib
			nickChanUsers := strings.SplitN(evt.Params, ":", 2)
			for _, u := range strings.Split(skipUserModeRepl.Replace(nickChanUsers[1]), " ") {
				fmt.Println("User: ", u)
			}
		}
	})

	return ic
}

func (ic *IrcConn) AddListener(l func(ic *IrcConn, evt *IrcEvent)) {
	ic.listeners = append(ic.listeners, l)
}

func (ic *IrcConn) Connected() bool {
	return ic.socket != nil
}

func (ic *IrcConn) SetNick(rc gocop.RunContext) (interface{}, error) {
	ic.nick = rc.Get("nick")
	if ic.Connected() {
		ic.SendRaw("NICK " + ic.nick)
	}
	return nil, nil
}

func (ic *IrcConn) SendRaw(data string) {
	if ic.Connected() {
		ic.write <- data + "\r\n"
	}
}

func (ic *IrcConn) Connect(rc gocop.RunContext) (interface{}, error) {
	nick := rc.Get("nick")
	user := rc.Get("user")
	if nick == "" {
		if ic.nick == "" {
			return nil, fmt.Errorf("Please set nick. Can add it after server on /connect")
		} else {
			nick = ic.nick
		}
	}
	if user == "" {
		user = nick
	}

	conn, err := net.DialTimeout("tcp", rc.Get("server"), time.Second*30)
	if err != nil {
		log.Panic(err)
	}

	ic.socket = conn

	ic.end = make(chan struct{})
	ic.write = make(chan string, 2)

	// Read loop
	go func() {
		buf := []byte{}
		reader := bufio.NewReader(ic.socket)
		for {
			select {
			case <-ic.end:
				return
			default:
				line, isPre, err := reader.ReadLine()
				if err != nil {
					log.Panic(err)
				}
				buf = append(buf, line...)
				if !isPre {
					ic.handleIncomming(string(buf))
					buf = buf[:0]
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ic.end:
				return
			case data := <-ic.write:
				ic.socket.Write([]byte(data))
				log.Print("<<< ", data)
			}
		}
	}()

	// Send out login....
	ic.SendRaw("NICK " + nick)
	ic.SendRaw("USER " + user + " 0.0.0.0 0.0.0.0 :" + nick)
	return "<connected>", nil
}

func (ic *IrcConn) handleIncomming(in string) {
	slice := []byte(in)
	tags := []byte{}
	prefix := []byte{}
	if slice[0] == '@' {
		tags, slice = splitTo(slice, ':')
	}
	if slice[0] == ':' {
		prefix, slice = splitTo(slice, ' ')
	}
	command, params := splitTo(slice, ' ')

	event := &IrcEvent{in, string(tags), string(prefix), string(command), string(params)}

	for _, l := range ic.listeners {
		l(ic, event)
	}
}

func main() {
	cp := gocop.NewCommandParser()
	irc := (&IrcConn{}).Init()

	world := cp.NewWorld()
	world.AddSubCommand("/nick").Handler(irc.SetNick).AddArgument("nick")
	world.AddSubCommand("/connect").Handler(irc.Connect).AddArgument("server").AddArgument("nick").Optional().AddArgument("user").Optional()
	world.AddSubCommand("/raw").AddArgument("data").Times(1, 999).Handler(func(rc gocop.RunContext) (res interface{}, err error) {
		irc.SendRaw(rc.Get("data"))
		return
	})

	world.AddSubCommand("/join").AddArgument("channel").Times(1, 2).Handler(func(rc gocop.RunContext) (res interface{}, err error) {
		irc.SendRaw("JOIN " + rc.Get("channel"))
		return
	})

	world.AddSubCommand("/msg").AddArgument("user").AddArgument("message").Times(1, 999).Handler(func(rc gocop.RunContext) (res interface{}, err error) {
		irc.SendRaw("PRIVMSG " + rc.Get("user") + " :" + rc.Get("message"))
		return
	})

	world.AddSubCommand("/list").AddArgument("channel").Handler(func(rc gocop.RunContext) (res interface{}, err error) {
		irc.SendRaw("LIST " + rc.Get("channel"))
		return
	})

	world.AddSubCommand("/who").AddArgument("channel").Handler(func(rc gocop.RunContext) (res interface{}, err error) {
		irc.SendRaw("WHO " + rc.Get("channel"))
		return
	})

	world.AddSubCommand("/whois").AddArgument("nick").Handler(func(rc gocop.RunContext) (res interface{}, err error) {
		irc.SendRaw("WHOIS " + rc.Get("nick"))
		return
	})

	world.AddSubCommand("/quit").Handler(func(rc gocop.RunContext) (res interface{}, err error) {
		irc.SendRaw("QUIT :" + rc.Get("message"))
		return
	}).AddArgument("message").Times(0, 999)

	log.Printf("Starting with PID %d, and parser %+v\n", os.Getpid(), cp)

	err := cp.MainLoop()

	log.Print("Exit main loop: ", err)

}
