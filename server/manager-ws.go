// =================================================================================
// Filename: manager-ws.go
// Function: manager websocket API handling
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2020-2022, 2025
// =================================================================================
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
func (cmd *Command) getManagerKeyFromURL(r *http.Request) (err error) {
	log.Println("i.getManagerKeyFromURL:", r.URL.Path)

	query := r.URL.Query()
	cmd.Key = query.Get("key")
	return
}

// ---------------------------------------------------------------------------------
func CheckManagerAccessPermission(r *http.Request) (err error) {
	log.Println("i.CheckManagerAccessPermission:", r.RemoteAddr)

	cmd := &Command{}
	err = cmd.getManagerKeyFromURL(r)
	if err != nil {
		log.Println(err)
		return
	}

	if mConfig.KeyManager != "" {
		if mConfig.KeyManager != cmd.Key {
			err = fmt.Errorf("invalid manager key: %s", cmd.Key)
			return
		}
	}
	if mConfig.HostCIDR != "" {
		if IsValidHostCIDR(mConfig.HostCIDR, r.RemoteAddr) {
			err = fmt.Errorf("invalid manager address: %s", r.RemoteAddr)
			return
		}
	}
	return
}

// ---------------------------------------------------------------------------------
func ProcManagerWsCommand(w http.ResponseWriter, r *http.Request) (err error) {
	log.Println("i.ProcManagerWSCommand:", r.URL)

	err = CheckManagerAccessPermission(r)
	if err != nil {
		log.Println(err)
		pStudio.pushEvent("mgr-in", "perm-error", err.Error())
		return
	}

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

	s := pStudio.addNewSessionWithNameRequest(r.URL.Path, "moth-manager")
	defer pStudio.deleteSession(s)

	pStudio.pushEvent("mgr-in", s.Name, s.ID)
	defer pStudio.pushEvent("mgr-out", s.Name, s.ID)

	for s.isState(Using) {
		cmd := NewCommandPointer()
		err = ws.ReadJSON(&cmd)
		if err != nil {
			log.Println("ws.ReadJSON:", err)
			break
		}

		str, err := cmd.execManager()
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
func ProcManagerWsEvent(w http.ResponseWriter, r *http.Request) (err error) {
	log.Println("i.ProcManagerWsEvent:", r.URL)

	ws, err := UpgradeToWebSocket(w, r, 1024)
	if err != nil {
		log.Println("UpgradeToWebSocket:", err)
		return
	}
	defer ws.Close()

	s := pStudio.addNewSessionWithNameRequest(r.URL.Path, "mgr-eventer")
	defer pStudio.deleteSession(s)

	pStudio.addEventer(s.ID, ws)
	defer pStudio.deleteEventer(s.ID)

	pStudio.pushEvent("evt-in", s.Name, s.ID)
	defer pStudio.pushEvent("evt-out", s.Name, s.ID)

	// --------------- keep the connection and check it
	for s.isState(Using) {
		time.Sleep(time.Second)
		err = ws.WriteMessage(websocket.PingMessage, []byte("eventer:ping"))
		if err != nil {
			log.Println(err)
			return
		}
		continue
	}
	return
}

// ---------------------------------------------------------------------------------
func StudioEventBroker(pst *Studio) (err error) {
	log.Println("IN StudioEventBroker:", pst.ID)
	defer log.Println("OUT StudioEventBroker:", pst.ID)

	w := pStudio.addNewWorkerWithParams("/manager/ws/evt", pst.ID, "system")
	defer pStudio.deleteWorker(w)

	pst.setEventState(Using)
	defer pst.setEventState(Idle)

	for w.isState(Using) {
		evt := <-pst.eventChan
		log.Println(evt)

		// send an event to all event receivers in studio
		for _, ews := range pst.Eventers {
			err = ews.WriteJSON(evt)
			if err != nil {
				log.Println(err)
				return
			}
		}

		// handle some specific events
		if evt.Name == "pub-in" {
			time.Sleep(time.Second)
			pst.checkBridges("auto")
		}
	}
	return
}

//=================================================================================
