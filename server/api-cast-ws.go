// =================================================================================
// Filename: api-cast-ws.go
// Function: simple cast(pub,sub only) websocket API for message based streaming
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021
// =================================================================================
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
func CastWSPublisher(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN CastWSPublisher:", qo.Source, qo.Track)
	defer log.Println("OUT CastWSPublisher:", err)

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	c := pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if c == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}
	c.AtUsed = time.Now()
	s.ChannelID = c.ID

	if c.Blocked || !c.isValidStreamKey(qo.Channel.Key) {
		err = fmt.Errorf("not allowed to use: %v, %s", c.Blocked, qo.Channel.Key)
		log.Println(err)
		return
	}

	_, trk, err := c.addSourceTrackByLabel(qo.Source.Label, qo.Track.Label)
	if err != nil {
		log.Println(err)
		return
	}

	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)

	err = trk.handleBuffersByCastAPI(ws, s)
	return
}

// ---------------------------------------------------------------------------------
// communication model : 1 -> N
func CastWSSubscriber(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN CastWSSubscriber:", qo.Source, qo.Track)
	defer log.Println("OUT CastWSSubscriber:", err)

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}
	s.ChannelID = s.chn.ID
	s.chn.AtUsed = time.Now()

	if s.chn.Blocked || !s.chn.isValidStreamKey(qo.Channel.Key) {
		err = fmt.Errorf("not allowed to use: %v, %s", s.chn.Blocked, qo.Channel.Key)
		log.Println(err)
		return
	}

	s.src, s.trk, err = s.chn.addSourceTrackByLabel(qo.Source.Label, qo.Track.Label)
	if err != nil {
		log.Println(err)
		return
	}

	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)

	err = s.trk.handleBuffersByCastAPI(ws, s)
	return
}

// ---------------------------------------------------------------------------------
func (trk *Track) handleBuffersByCastAPI(ws *websocket.Conn, s *Session) (err error) {
	log.Println("i.handleBuffersByCastAPI:", s.Name)
	defer log.Println("o.handleBuffersByCastAPI:", err)

	switch s.Name {
	case "/cast/ws/pub":
		rbuf := trk.Rings[BUFFER_NUM_FORE] // 0: foreward direction
		err = rbuf.recvTrackBufferInWSMessage(ws, s, true)
	case "/cast/ws/sub":
		sbuf := trk.Rings[BUFFER_NUM_FORE] // 0: forward direction
		err = sbuf.sendTrackBufferInWSMessage(ws, s, true)
	default:
		err = fmt.Errorf("not support API: %s", s.Name)
	}
	return
}

//=================================================================================
