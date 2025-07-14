// =================================================================================
// Filename: monitor-http.go
// Function: processing user control commands
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-2022
// =================================================================================
package main

import (
	"io"
	"log"
	"net/http"
)

// ---------------------------------------------------------------------------------
func ProcMonitorHTTPCommand(w http.ResponseWriter, r *http.Request) (str string, err error) {
	log.Println("i.ProcMonitorHTTPCommand:")
	defer log.Println("o.ProcMonitorHTTPCommand:", err)

	// s := pStudio.addNewSessionWithName("/monitor/http/cmd")
	// defer pStudio.deleteSession(s)

	cmd := NewCommandPointer()
	defer func() {
		SendHTTPResponse(w, cmd.Format, str, err)
	}()

	cmd.Path = r.URL.Path
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	cmd.Data = string(body)

	err = cmd.parseQuery(r)
	if err != nil {
		log.Println("cmd.parseQuery:", err)
		return
	}
	log.Println(cmd)

	err = cmd.checkMonitorPermission()
	if err != nil {
		log.Println("cmd.checkMonitorPermission:", err)
		return
	}

	str, err = cmd.execMonitor()
	if err != nil {
		log.Println("cmd.execMonitor:", err)
		return
	}
	return
}

//=================================================================================
