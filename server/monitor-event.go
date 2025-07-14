// =================================================================================
// Filename: monitor-event.go
// Function: monitor event messages
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-2022
// =================================================================================
package main

import (
	"log"
	"net/http"
	"time"
)

// ---------------------------------------------------------------------------------
func ProcMonitorWSEvent(w http.ResponseWriter, r *http.Request) (err error) {
	log.Println("i.ProcMonitorWSEvent:")

	ws, err := UpgradeToWebSocket(w, r, 1024)
	if err != nil {
		log.Println("UpgradeToWebSocket:", err)
		return
	}
	defer ws.Close()

	ws.SetPingHandler(func(msg string) (err error) {
		ws.SetReadDeadline(time.Now().Add(TIME_WAIT_BASE_SECONDS))
		return
	})

	s := pStudio.addNewSessionWithName(r.URL.Path)
	defer pStudio.deleteSession(s)

	for s.isState(Using) {
		select {
		case <-pStudio.eventChan:
			// send event to all subscribers
		default:
			time.Sleep(1 * time.Second)
		}
	}
	return
}

//=================================================================================
