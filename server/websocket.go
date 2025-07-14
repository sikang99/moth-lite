// =================================================================================
// Filename: websocket.go
// Function: websocket data & functions
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021
// https://github.com/eranyanay/1m-go-websockets
// =================================================================================
package main

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
const (
	TimeCheckBaseInterval    = 30 * time.Second
	TimeCheckPingInterval    = 10 * time.Second
	TimeCheckSessionInterval = 3 * time.Second
	TimeCheckLoopInterval    = 1 * time.Second
)

// ---------------------------------------------------------------------------------
func UpgradeToWebSocket(w http.ResponseWriter, r *http.Request, bsize int) (ws *websocket.Conn, err error) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  bsize, // 1024
		WriteBufferSize: bsize, // 1024
		// EnableCompression: true, // experimental
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	ws, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrader.Upgrade:", err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
func ConnectToWebSocket(url string, bsize int) (ws *websocket.Conn, err error) {
	var dialer = websocket.Dialer{
		ReadBufferSize:    bsize, // 1024,
		WriteBufferSize:   bsize, // 1024,
		EnableCompression: true,
		Proxy:             http.ProxyFromEnvironment,
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
	}

	ws, _, err = dialer.Dial(url, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// --- Basic style
	// ws, _, err = websocket.DefaultDialer.Dial(url, nil)
	// if err != nil {
	// 	log.Println(err)
	// 	return
	// }

	return
}

// ---------------------------------------------------------------------------------
func EchoWSMessage(ws *websocket.Conn, style string) (err error) {
	log.Println("i.EchoWSMessage:", style)

	switch style {
	case "normal":
		mt, p, err := ws.ReadMessage()
		if err != nil {
			log.Println(err)
			return err
		}
		err = ws.WriteMessage(mt, p)
		if err != nil {
			log.Println(err)
			return err
		}
	case "copy":
		mt, rd, err := ws.NextReader()
		if err != nil {
			log.Println(err)
			return err
		}
		wr, err := ws.NextWriter(mt)
		if err != nil {
			log.Println(err)
			return err
		}
		n, err := io.Copy(wr, rd)
		if err != nil {
			log.Println(err, n)
			return err
		}
		wr.Close()
	case "json":
		m := &SignalMessage{}
		err = ws.ReadJSON(m)
		if err != nil {
			log.Println(err)
			return
		}
		err = ws.WriteJSON(m)
		if err != nil {
			log.Println(err)
			return
		}
	}
	return
}

//=================================================================================
