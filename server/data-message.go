// =================================================================================
// Filename: data-message.go
// Function: message handing functions
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021
// =================================================================================
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
type WSMessage struct {
	Type      string    `json:"type"`
	Data      string    `json:"data,omitempty"`
	Name      string    `json:"name,omitempty"`
	Path      string    `json:"path,omitempty"`
	RequestID string    `json:"req_id,omitempty"`
	AtCreated time.Time `json:"at_created,omitempty"`
}

//---------------------------------------------------------------------------------
// func (s *Session) messageReceiver(ws *websocket.Conn, c *Channel) {

// 	for s.isUsing() && c.isUsing() {
// 		rm := WSMessage{}
// 		select {
// 		case rm = <-s.MsgChan:
// 		case rm = <-c.MsgChan:
// 		}
// 		s.procControlMessage(ws, &rm, c)
// 	}
// }

// ---------------------------------------------------------------------------------
func (s *Session) procControlMessage(ws *websocket.Conn, rm *WSMessage) (err error) {
	// log.Println("i.procControlMessage:", rm.Type)

	sm := &WSMessage{}
	defer func() {
		if err != nil {
			sm.Type = "error"
			sm.Data = err.Error()
			log.Println(rm.Type, err)
		}
		err = ws.WriteJSON(sm)
	}()

	qo := &QueryOption{}
	err = json.Unmarshal([]byte(rm.Data), qo)
	if err != nil {
		return
	}

	switch rm.Type {
	case "ping":
		sm.Type = "pong"
	case "info_channel":
		sm.Type = "channel"
		data, err := json.Marshal(s.chn)
		if err != nil {
			return err
		}
		sm.Data = string(data)
	case "info_source":
		sm.Type = "source"
		src := s.chn.findSourceByLabel(qo.Source.Label)
		if src == nil {
			err = fmt.Errorf("not found source: %s", qo.Source.Label)
			return
		}
		data, err := json.Marshal(src)
		if err != nil {
			return err
		}
		sm.Data = string(data)
	case "info_track":
		sm.Type = "track"
		s.src, s.trk, _ = s.chn.findSourceTrackByLabel(qo.Source.Label, qo.Track.Label)
		if s.trk == nil {
			err = fmt.Errorf("not found source/track: %s/%s", qo.Source.Label, qo.Track.Label)
			return
		}
		data, err := json.Marshal(s.trk)
		if err != nil {
			return err
		}
		sm.Data = string(data)
	case "set_buffer":
		sm.Type = "buffer"
		// TBD: set_buffer
		// buf = trk[qo.Buffer.Order]
		// buf.Len = qo.Buffer.Len
	case "set_channel":
		sm.Type = "channel"
		if qo.Channel.Key != "" {
			if qo.Channel.Key == "off" {
				s.chn.StreamKey = ""
			} else {
				s.chn.StreamKey = qo.Channel.Key
			}
		}
		if qo.Channel.Record != "" {
			if qo.Channel.Record == "off" {
				s.chn.RecordAuto = false
			}
			if qo.Channel.Record == "on" {
				s.chn.RecordAuto = true
			}
		}
		if qo.Channel.Trans != "" {
			if qo.Channel.Trans == "off" {
				s.chn.TransAuto = false
			}
			if qo.Channel.Trans == "on" {
				s.chn.TransAuto = true
			}
		}
		data, err := json.Marshal(s.chn)
		if err != nil {
			return err
		}
		sm.Data = string(data)
	case "close_channel":
		sm.Type = "channel"
		if !IsXidString(qo.Channel.ID) {
			err = fmt.Errorf("invalid channel ID: %s", qo.Channel.ID)
			return
		}
		if s.chn.ID == qo.Channel.ID && s.chn.isState(Using) {
			s.chn.State = Idle
		}
		data, err := json.Marshal(s.chn)
		if err != nil {
			return err
		}
		sm.Data = string(data)
	case "show_session":
		sm.Type = "session"
		list := pStudio.listSessionsByChannelIDSessionName(s.chn.ID, qo.Session.Name)
		if len(list) == 0 {
			err = fmt.Errorf("no session for %s", qo.Session.Name)
			return
		}
		data, err := json.Marshal(list)
		if err != nil {
			return err
		}
		sm.Data = string(data)
	case "info_session": // my_session
		sm.Type = "session"
		data, err := json.Marshal(s)
		if err != nil {
			return err
		}
		sm.Data = string(data)
	case "close_session":
		sm.Type = "session"
		if !IsXidString(qo.Session.ID) {
			err = fmt.Errorf("invalid session ID: %s", qo.Session.ID)
			return
		}
		ses := pStudio.findSessionByChannelIDSessionID(s.chn.ID, qo.Session.ID)
		if ses == nil {
			err = fmt.Errorf("not found session: %s", qo.Session.ID)
			return
		}
		if ses.isState(Using) {
			ses.State = Idle
		}
		data, err := json.Marshal(ses)
		if err != nil {
			return err
		}
		sm.Data = string(data)
	default:
		err = fmt.Errorf("unknown message type: %s", rm.Type)
	}
	return
}

// ---------------------------------------------------------------------------------
func SafeCloseMessage(ch chan EventMessage) (closed bool) {
	defer func() {
		if recover() != nil {
			// The return result can be altered
			// in a defer function call.
			closed = false
		}
	}()

	// assume ch != nil here.
	close(ch)   // panic if ch is closed
	return true // <=> justClosed = true; return
}

func SafeSendMessage(ch chan EventMessage, value EventMessage) (closed bool) {
	defer func() {
		if recover() != nil {
			closed = true
		}
	}()

	ch <- value  // panic if ch is closed
	return false // <=> closed = false; return
}

//=================================================================================
