// =================================================================================
// Filename: monitor-ws.go
// Function: processing user control commands
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021
// =================================================================================
package main

import (
	"log"
	"net/http"
	"time"
)

// ---------------------------------------------------------------------------------
func ProcMonitorWSCommand(w http.ResponseWriter, r *http.Request) (err error) {
	log.Println("i.ProcMonitorWSCommand:")

	ws, err := UpgradeToWebSocket(w, r, 1024)
	if err != nil {
		log.Println("UpgradeToWebSocket:", err)
		return
	}
	defer ws.Close()

	// extend the websocket timeout
	ws.SetPingHandler(func(msg string) (err error) {
		ws.SetReadDeadline(time.Now().Add(TIME_WAIT_BASE_SECONDS))
		log.Println("[PING]", msg)
		return
	})

	s := pStudio.addNewSessionWithName(r.URL.Path)
	defer pStudio.deleteSession(s)

	for s.isState(Using) {
		cmd := &Command{}
		err = ws.ReadJSON(&cmd)
		if err != nil {
			log.Println("ws.ReadJSON:", err)
			break
		}

		err = cmd.checkMonitorPermission()
		if err != nil {
			log.Println("cmd.checkMonitorPermission:", err)
			break
		}

		str, err := cmd.execMonitor()
		if err != nil {
			log.Println(err)
			break
		}
		cmd.Data = str

		err = ws.WriteJSON(cmd)
		if err != nil {
			log.Println("ws.WriteJSON:", err)
			break
		}
	}
	return
}

// ---------------------------------------------------------------------------------
func ProcControlWSCommand(w http.ResponseWriter, r *http.Request) (err error) {
	log.Println("i.ProcControlWSCommand:")
	defer log.Println("o.ProcControlWSCommand:", err)

	ws, err := UpgradeToWebSocket(w, r, 1024)
	if err != nil {
		log.Println("UpgradeToWebSocket:", err)
		return
	}
	defer ws.Close()

	// TBD
	return
}

//=================================================================================
