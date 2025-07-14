// =================================================================================
// Filename: monitor-cmd.go
// Function: command processing for control api
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-2023
// =================================================================================
package main

import (
	"fmt"
	"log"
)

// ---------------------------------------------------------------------------------
func (cmd *Command) checkMonitorPermission() (err error) {
	log.Println("i.checkMonitorPermission:", cmd.Op, cmd.Obj)

	switch cmd.Op {
	case "ping", "show", "check":
		if cmd.Obj == "config" {
			err = fmt.Errorf("%s is not allowed for %s", cmd.Op, cmd.Obj)
			return
		}
	default:
		err = fmt.Errorf("%s is not allowed for %s api", cmd.Op, cmd.Path)
	}
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) execMonitor() (str string, err error) {
	// log.Println("execMonitor:", cmd.Op)

	switch cmd.Op {
	case "ping":
		str = "Pong!"
		if cmd.Format == "json" {
			str = fmt.Sprintf("{ \"result\": \"%s\" }", str)
		}
	case "show":
		switch cmd.Obj {
		case "session":
			if cmd.ID == "" {
				str, err = cmd.showSessions()
			} else {
				str, err = cmd.infoSession()
			}
		case "channel":
			if cmd.ID == "" {
				str, err = cmd.showChannels()
			} else {
				str, err = cmd.infoChannel()
			}
		default:
			err = fmt.Errorf("invalid object: %s", cmd.Obj)
		}
	case "check":
		switch cmd.Obj {
		case "channel":
			str, err = cmd.checkChannelResource()
		default:
			err = fmt.Errorf("invalid object: %s", cmd.Obj)
		}
	default:
		err = fmt.Errorf("not supported op %s", cmd.Op)
	}
	return
}

//=================================================================================
