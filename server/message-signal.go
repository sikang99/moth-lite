// =================================================================================
// Filename: message-signal.go
// Function: Signalling server for moth
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021
// =================================================================================
package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
type SignalMessage struct {
	Type  string      `json:"type"`
	Data  string      `json:"data,omitempty"`  // json data string
	Block interface{} `json:"block,omitempty"` // block, any data, for custom use
}

func (d *SignalMessage) String() (str string) {
	str = fmt.Sprintf("Type: %s, Data: %s", d.Type, d.Data)
	return
}

// ---------------------------------------------------------------------------------
type SignalClient struct {
	sync.Mutex
	Common `json:"common"`
	Custom interface{} `json:"custom,omitempty"`
}

func (d *SignalClient) String() (str string) {
	str = d.Common.String()
	return
}

func NewSignalClient(name string) (d *SignalClient) {
	d = &SignalClient{}
	d.ID = GetXidString()
	d.Name = name
	return
}

// ---------------------------------------------------------------------------------
type SignalGroup struct {
	sync.Mutex
	Common    `json:"common"`
	Clients   map[*websocket.Conn]*SignalClient `json:"clients,omitempty"`
	EventChan chan SignalMessage
}

func (d *SignalGroup) String() (str string) {
	str = d.Common.String()
	str += "\n\tClients: "
	for _, v := range d.Clients {
		str += fmt.Sprintf("%s: %s, ", v.ID, v.Name)
	}
	return
}

func NewSignalGroup(name string) (d *SignalGroup) {
	d = &SignalGroup{
		Clients: make(map[*websocket.Conn]*SignalClient),
	}
	d.ID = GetXidString()
	d.Name = name
	return
}

func (d *SignalGroup) findClient(ws *websocket.Conn) (r *SignalClient) {
	d.Lock()
	defer d.Unlock()
	r = d.Clients[ws]
	return
}

func (d *SignalGroup) addClient(ws *websocket.Conn, sc *SignalClient) (r *SignalClient) {
	d.Lock()
	defer d.Unlock()
	d.Clients[ws] = sc
	return
}

func (d *SignalGroup) deleteClient(ws *websocket.Conn) {
	d.Lock()
	defer d.Unlock()
	delete(d.Clients, ws)
	return
}

// ---------------------------------------------------------------------------------
type SignalCenter struct {
	sync.RWMutex
	Common `json:"common"`
	Groups map[string]*SignalGroup `json:"groups,omitempty"`
}

func (d *SignalCenter) String() (str string) {
	str = d.Common.String()
	str += "\n\tGroups: "
	for _, v := range d.Groups {
		str += fmt.Sprintf("%s: %s, ", v.ID, v.Name)
	}
	return
}

func NewSignalCenter(name string) (d *SignalCenter) {
	d = &SignalCenter{
		Groups: make(map[string]*SignalGroup),
	}
	d.ID = GetXidString()
	d.Name = name
	return
}

func (d *SignalCenter) findGroup(id string) (r *SignalGroup) {
	d.Lock()
	defer d.Unlock()
	r = d.Groups[id]
	return
}

func (d *SignalCenter) addGroup(sc *SignalGroup) (r *SignalGroup) {
	d.Lock()
	defer d.Unlock()
	d.Groups[sc.ID] = sc
	return
}

func (d *SignalCenter) deleteGroup(id string) {
	d.Lock()
	defer d.Unlock()
	delete(d.Groups, id)
	return
}

// ---------------------------------------------------------------------------------
func ProcSignalMessage(rm, sm *SignalMessage) (err error) {

	switch rm.Type {
	case "ping":
		sm.Type = "pong"
	case "register":
		sm.Type = "client"
	default:
		err = fmt.Errorf("unknown message type: %s", rm.Type)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
var sigGroup = &SignalGroup{
	Clients: make(map[*websocket.Conn]*SignalClient),
}

// var sigCenter = &SignalCenter{
// 	Groups: make(map[string]*SignalGroup),
// }

// ---------------------------------------------------------------------------------
func SignalMothWSClient(url string) (err error) {
	log.Println("IN SignalWSClient:", url)
	defer log.Println("OUT SignalWSClient:", url, err)

	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	for {
		sm := &SignalMessage{Type: "ping"}
		err = ws.WriteJSON(sm)
		if err != nil {
			log.Println(err)
			return
		}

		rm := &SignalMessage{}
		err = ws.WriteJSON(rm)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(rm.Type)

		time.Sleep(3 * time.Second)
	}
}

// ---------------------------------------------------------------------------------
func SignalWSServer(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN SignalWSServer:", qo.Source, qo.Track)
	defer log.Println("OUT SignalWSServer:", err)

	client := &SignalClient{}
	client.ID = GetXidString()
	client.Name = time.Now().Format("20060102150405")

	sigGroup.Clients[ws] = client
	defer func() {
		delete(sigGroup.Clients, ws)
	}()
	log.Println(sigGroup)

	for {
		rm := &SignalMessage{}
		err = ws.ReadJSON(rm)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(rm)

		sm := &SignalMessage{}
		err = ProcSignalMessage(rm, sm)
		if err != nil {
			log.Println(err)
			return
		}

		err = ws.WriteJSON(sm)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

// ---------------------------------------------------------------------------------
func SignalWSBroker(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN SignalWSBroker:", qo.Source, qo.Track)
	defer log.Println("OUT SignalWSBroker:", err)

	sigGroup.ID = GetXidString()
	sigGroup.Name = "Sample Group"

	return
}

//=================================================================================
