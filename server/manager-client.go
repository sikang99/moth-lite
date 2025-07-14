// =================================================================================
// Filename: manager-client.go
// Function: Server manager client function
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2020-2023
// =================================================================================
package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
// HTTP based manager client
// ---------------------------------------------------------------------------------
func StartHTTPManagerClient(url, key string) (err error) {
	// log.Println("IN StartHTTPManagerClient:", url)

	var precmd Command

	for {
		cmd, err := ReadManagerCommand(url, key)
		if err != nil {
			log.Println(err)
			continue
		}

		if cmd.Op == "previous" {
			cmd = precmd
		} else {
			precmd = cmd
		}

		err = SendRecvHTTPCommand(cmd)
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

// ---------------------------------------------------------------------------------
func PrintNewID() {
	PrintXidStringInfo(GetXidString())
}

// ---------------------------------------------------------------------------------
func PrintManagerHelpMessage() {
	fmt.Printf("%s\n", mConfig.ProgramTitle)
	fmt.Printf("\nusage: op obj [id|name] [format]")
	fmt.Printf("\n\top     = [show|set|run|add|delete|load|save], [xid|ping|help|quit]")
	fmt.Printf("\n\tobj    = [config:0|session:1|channel:2|group:4|bridge:5|punch:6|ticket:7|worker:8|studio:9]")
	fmt.Printf("\n\tid     = <xid>")
	fmt.Printf("\n\topt    = [state|name|key|record|trans|procs|relay|path|source|track|auto]")
	fmt.Printf("\n\tstate  = [idle|using|block|on|off|start|stop]")
	fmt.Printf("\n\tstyle  = [static|instant|dynamic]")
	fmt.Printf("\n\tformat = [json|text|mp4|webm]")
	fmt.Printf("\n\tvalue  = <string> or <integer>, ex) Kang, 1234")
	fmt.Printf("\n\tExample : command> show session bq6m79ug10l3ebijndhg")
	fmt.Printf("\n\tShortcut: 1) show session, 2) show channel, 4) show group, 5) show bridge, 6) show punch, 0) show config")
	fmt.Printf("\n\t          show channel 21) using, 22) <xid>, 23) <xid> track, 24) <xid> record, 25/26) <xid> trans")
	fmt.Printf("\n")
}

// ---------------------------------------------------------------------------------
func ReadManagerCommand(url, key string) (cmd Command, err error) {
	// log.Println("i.ReadMonitorCommand:")

	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("command> ")
		sc.Scan()
		cmdstr := sc.Text()
		if len(cmdstr) == 0 {
			continue
		}

		// !(33), shortcut for "run shell"
		feelmark := cmdstr[0]
		if feelmark == 33 {
			cmdstr = fmt.Sprintf("run shell %s", cmdstr[1:])
		}

		switch cmdstr {
		case ".": // repeat the previous command
			cmd.Op = "previous"
			return
		case "0", "cfg", "config":
			cmdstr = "show config"
		case "1", "ses", "session":
			cmdstr = "show session"
		case "2", "chn", "channel":
			cmdstr = "show channel"
		case "21":
			cmdstr = "show channel using"
		case "22":
			cmdstr = fmt.Sprintf("show channel %s", "c40hp6epjh65aeq6ne50")
		case "23":
			cmdstr = fmt.Sprintf("show channel %s track", "c40hp6epjh65aeq6ne50")
		case "24":
			cmdstr = fmt.Sprintf("run channel %s record", "c40hp6epjh65aeq6ne50")
		case "25":
			cmdstr = fmt.Sprintf("run channel %s trans %s", "c40hp6epjh65aeq6ne50", "vp8")
		case "26":
			cmdstr = fmt.Sprintf("run channel %s trans %s", "c40hp6epjh65aeq6ne50", "jpeg")
		case "27":
			cmdstr = fmt.Sprintf("check channel %s %s", "c40hp6epjh65aeq6ne50", "base/video")
		case "4", "grp", "group":
			cmdstr = "show group"
		case "5", "brd", "bridge":
			cmdstr = "show bridge"
		case "51":
			cmdstr = "show bridge using"
		case "52":
			cmdstr = fmt.Sprintf("show bridge %s", "cg41tpiuab7udhkv4nrg")
		case "53":
			cmdstr = fmt.Sprintf("run bridge %s start", "cg41tpiuab7udhkv4nrg")
		case "54":
			cmdstr = fmt.Sprintf("run bridge %s stop", "cg41tpiuab7udhkv4nrg")
		case "6", "pnh", "punch":
			cmdstr = "show punch"
		case "7", "tkt", "ticket":
			cmdstr = "show ticket"
		case "8", "wrk", "worker":
			cmdstr = "show worker"
		case "id", "xid":
			PrintNewID()
			continue
		case "?", "help", "usage":
			PrintManagerHelpMessage()
			continue
		case "quit", "exit":
			fmt.Println("Bye bye!")
			os.Exit(0)
			return
		}

		cmd, err = ParseManagerCommand(cmdstr, url, key)
		if err != nil {
			fmt.Println(err)
			continue
		}
		break
	}
	return
}

// ---------------------------------------------------------------------------------
func ParseManagerCommand(cmdstr, base, key string) (cmd Command, err error) {
	// log.Println("i.parseMonitorCommand:", cmdstr)

	toks := strings.Fields(cmdstr)
	// log.Println(len(toks))

	// the first operation type
	if len(toks) < 1 {
		err = fmt.Errorf("> <op> is required")
		return
	}

	cmd.Op = toks[0]
	switch toks[0] {
	case "show", "set", "run", "add", "delete", "check", "clean", "load", "save":
		toks = append(toks[:0], toks[1:]...) // delete op part
		// the second object type is required
		if len(toks) < 1 {
			err = fmt.Errorf("> <obj> is required")
			return
		}
	case "ping":
		goto SEND_COMMAND
	default:
		err = fmt.Errorf("> valid <op> is required: %s", toks[0])
		return
	}

	switch toks[0] {
	case "session", "channel", "room", "group", "punch", "ticket", "worker",
		"bridge", "route", "studio", "theater", "config", "system":
		cmd.Obj = toks[0]
		toks = append(toks[:0], toks[1:]...) // delete obj part from token slice
	case "shell":
		cmd.Obj = toks[0]
		toks = append(toks[:0], toks[1:]...) // delete obj part
		for i, tok := range toks {           // gather the remaining parts into a string
			if i == 0 {
				cmd.Value += tok
			} else {
				cmd.Value += fmt.Sprintf(" %s", tok)
			}
		}
		goto SEND_COMMAND
	default:
		err = fmt.Errorf("> valid <obj> is required: %s", toks[0])
		return
	}

	// the remaining parts : option, id, format, name
	for i := range toks {
		switch toks[i] {
		case "idle", "using", "close", "on", "off", "start", "stop":
			cmd.State = toks[i]
		case "static", "dynamic", "instant":
			cmd.Style = toks[i]
		case "state", "block", "path", "source", "track", "actor", "name", "style", "key", "disk", "record", "trans", "procs", "relay", "auto":
			cmd.Opt = toks[i]
		case "text", "json", "xml", "mp4", "webm":
			cmd.Format = toks[i]
		default:
			if IsXidString(toks[i]) {
				cmd.ID = toks[i]
			} else { // [<string>|<number>]
				cmd.Value = toks[i]
			}
		}
	}

	// check if the command and its params are all valid
	switch cmd.Op {
	case "show":
		if cmd.Obj == "" {
			err = fmt.Errorf("> %s: <obj> is required", cmd.Op)
			return
		}
	case "add":
		if cmd.Obj == "" || cmd.Value == "" {
			err = fmt.Errorf("> %s: <obj,name> are required", cmd.Op)
			return
		}
	case "delete":
		if cmd.Obj == "" || cmd.ID == "" {
			err = fmt.Errorf("> %s: <obj,id> are required", cmd.Op)
			return
		}
	case "clean", "load", "save":
		if cmd.Obj == "" {
			err = fmt.Errorf("> %s: <obj> is required", cmd.Op)
			return
		}
	case "set":
		if cmd.Obj == "" || cmd.ID == "" {
			err = fmt.Errorf("> %s: <obj,id,opt|state> are required", cmd.Op)
			return
		}
	case "run":
		if cmd.Obj == "" || cmd.ID == "" {
			err = fmt.Errorf("> %s: <obj,id> are required", cmd.Op)
			return
		}
	case "check":
		if cmd.Obj == "" || cmd.ID == "" || cmd.Opt == "" {
			err = fmt.Errorf("> %s: <obj,id,opt> are required", cmd.Op)
			return
		}
	case "ping":
	default:
		err = fmt.Errorf("> %s: invalid command", cmdstr)
		return
	}

SEND_COMMAND:
	// make a query url for the command
	cmd.Method = "POST"
	params := url.Values{}
	if cmd.Op != "" {
		params.Add("op", cmd.Op)
	}
	if cmd.Obj != "" {
		params.Add("obj", cmd.Obj)
	}
	if cmd.ID != "" {
		params.Add("id", cmd.ID)
	}
	if cmd.Name != "" {
		params.Add("name", cmd.Name)
	}
	if cmd.State != "" {
		params.Add("state", cmd.State)
	}
	if cmd.Style != "" {
		params.Add("style", cmd.Style)
	}
	if cmd.Opt != "" {
		params.Add("opt", cmd.Opt)
	}
	if cmd.Value != "" {
		params.Add("value", cmd.Value)
	}
	if cmd.Format != "" {
		params.Add("format", cmd.Format)
	}
	if key != "" { // manager's key
		params.Add("key", key)
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		err = fmt.Errorf("> url.Parse error: %s", base)
		return
	}

	baseURL.RawQuery = params.Encode()
	cmd.url = fmt.Sprint(baseURL)
	return
}

// ---------------------------------------------------------------------------------
func SendRecvHTTPCommand(cmd Command) (err error) {
	// log.Println("i.SendRecvHTTPCommand:")
	var res *http.Response

	// check the input params
	if cmd.Method == "" || cmd.url == "" {
		err = fmt.Errorf("> insufficient op params")
		return
	}

	// To ignore error 'x509: certificate signed by unknown authority'
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// send the http request
	if cmd.Method == "POST" {
		res, err = client.Post(cmd.url, "application/json", bytes.NewBuffer([]byte(cmd.Data)))
	} else {
		res, err = client.Get(cmd.url)
	}
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()

	// recv its response
	body, err := io.ReadAll(res.Body)
	fmt.Printf("%s\n", string(body))
	return
}

// ---------------------------------------------------------------------------------
// Websocket based manager client
// ---------------------------------------------------------------------------------
func StartWSManagerClient(url, key string) (err error) {
	// log.Println("IN StartWSManagerClient:", url, key)

	url += fmt.Sprintf("?key=%s", key)
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	ticker := time.NewTicker(TimeCheckPingInterval)
	defer ticker.Stop()

	cmdch := make(chan Command, 2)
	go ReadManagerCommandToChan(url, cmdch)

	for {
		select {
		case cmd := <-cmdch:
			err = SendRecvWSCommand(ws, cmd)
			if err != nil {
				log.Println(err)
				return
			}
		case <-ticker.C:
			err = ws.WriteMessage(websocket.PingMessage, []byte("manager:ping"))
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------------
func SendRecvWSCommand(ws *websocket.Conn, cmd Command) (err error) {
	// log.Println("i.SendRecvWSCommand:")

	err = ws.WriteJSON(cmd)
	if err != nil {
		log.Println(err)
		return
	}

	err = ws.ReadJSON(&cmd)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println(cmd.Data)
	return
}

// ---------------------------------------------------------------------------------
// Websocket based control client
// ---------------------------------------------------------------------------------
func StartWSControlClient(url, channel string) (err error) {
	// log.Println("IN StartWSControlClient:", url)

	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	for {
		qo := &QueryOption{}
		qo.Channel.ID = channel

		sm, err := ReadControlWSMessage(ws, qo)
		if err != nil {
			log.Println(err)
			break
		}

		rm, err := SendRecvWSMessage(ws, sm)
		if err != nil {
			log.Println(err)
			break
		}
		fmt.Println("result:", "(type)", rm.Type, "(data)", rm.Data)
	}
	return
}

// ---------------------------------------------------------------------------------
func ReadControlWSMessage(ws *websocket.Conn, qo *QueryOption) (sm *WSMessage, err error) {
	// log.Println("i.ReadControlWSMessage:")

	sc := bufio.NewScanner(os.Stdin)
	for {
		fmt.Printf("command> ")
		sc.Scan()
		str := sc.Text()
		if len(str) == 0 {
			continue
		}
		toks := strings.Fields(str)
		if len(toks) == 0 {
			continue
		}

		switch toks[0] {
		case "info_channel":
		case "close_channel":
			if len(toks) > 1 {
				qo.Channel.ID = toks[1]
			}
		case "set_channel":
			if len(toks) < 3 {
				fmt.Println("> set_channel [name|key|record|trans] [<string>|on|off]")
				continue
			}
			switch toks[1] {
			case "name":
				qo.Channel.Name = toks[2]
			case "key":
				qo.Channel.Key = toks[2]
			case "record":
				qo.Channel.Record = toks[2]
			case "trans":
				qo.Channel.Trans = toks[2]
			default:
				continue
			}
		case "info_source":
			qo.Source.Label = "base"
			if len(toks) > 1 {
				qo.Source.Label = toks[1]
			}
		case "info_track":
			qo.Source.Label = "base"
			if len(toks) > 1 {
				qo.Source.Label = toks[1]
			}
			qo.Track.Label = "video"
			if len(toks) > 2 {
				qo.Track.Label = toks[2]
			}
		case "set_buffer":
			// TBD: set_buffer
		case "info_session":
		case "show_session":
			if len(toks) > 1 {
				qo.Session.Name = toks[1]
			}
		case "close_session":
			if len(toks) > 1 {
				qo.Session.ID = toks[1]
			}
		case "ping":
		case "?", "help":
			fmt.Println("usage: [command] [options]")
			fmt.Println("\t- info_channel: info for my channel information")
			fmt.Println("\t- set_channel: set my channel information for [name|key|record|trans]")
			fmt.Println("\t- close_channel: close my channel")
			fmt.Println("\t- info_source: info for source information of my channel")
			fmt.Println("\t- info_track: info for track information of my channel source")
			fmt.Println("\t- show_session: list sessions in my channel")
			fmt.Println("\t- info_session: info for the session in my channel")
			fmt.Println("\t- close_session: close the session in my channel")
			fmt.Println("\t- ping: send ping and receive pong")
			fmt.Println("\t- help: this message")
			continue
		case "exit", "quit":
			fmt.Println("Bye bye!")
			os.Exit(0)
		default:
			fmt.Println("> not support operation:", toks[0])
			continue
		}

		sm = &WSMessage{}
		data, _ := json.Marshal(qo)
		sm.Type = toks[0]
		sm.Data = string(data)
		break
	}
	return
}

// ---------------------------------------------------------------------------------
func SendRecvWSMessage(ws *websocket.Conn, sm *WSMessage) (rm *WSMessage, err error) {
	// log.Println("i.SendRecvWSMessage:")

	err = ws.WriteJSON(sm)
	if err != nil {
		log.Println(err)
		return
	}
	// log.Println("[Tx]", sm.Type, sm.Data)

	rm = &WSMessage{}

	err = ws.ReadJSON(rm)
	if err != nil {
		log.Println(err)
		return
	}
	// log.Println("[Rx]", rm.Type, rm.Data)
	return
}

//=================================================================================
