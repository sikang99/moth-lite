// =================================================================================
// Filename: monitor-client.go
// Function: Server manager client function
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2020-2022
// =================================================================================
package main

import (
	"log"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
type Program struct {
	ok bool
}

// ---------------------------------------------------------------------------------
// Websocket based monitor client
// ---------------------------------------------------------------------------------
func StartWSMonitorClient(url string) (err error) {
	// log.Println("IN StartWSManagerClient:", url)

	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

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
		case <-time.After(TimeCheckPingInterval):
			err = ws.WriteMessage(websocket.PingMessage, []byte("monitor:ping"))
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

// ---------------------------------------------------------------------------------
func ReadManagerCommandToChan(url string, cmdch chan Command) (err error) {
	// log.Println("IN ReadManagerCommandToChan:", url)

	var precmd Command

	for {
		cmd, err := ReadManagerCommand(url, "")
		if err != nil {
			log.Println(err)
			continue
		}

		if cmd.Op == "previous" {
			cmd = precmd
		} else {
			precmd = cmd
		}

		cmdch <- cmd
	}
}

//=================================================================================
