// =================================================================================
// Filename: api-pang-ws.go
// Function: pang HTTP API for websocket message based streaming
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-2025
// =================================================================================
package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
// API: /pang/ws/eco, communication model : 1 <-> 0
// ---------------------------------------------------------------------------------
func PangWSReflector(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangWSReflector:", qo.Channel, qo.Session.Unit)
	defer log.Println("OUT PangWSReflector:", err)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)

	for s.isState(Using) {
		mt, data, err := ws.ReadMessage()
		if err != nil {
			log.Println("ws.ReadMessage:", err)
			break
		}
		s.InBytes += len(data)

		// simulate the buffering delay
		time.Sleep(s.TimeUnit)

		err = ws.WriteMessage(mt, data)
		if err != nil {
			log.Println("ws.WriteMessage:", err)
			break
		}
		s.OutBytes += len(data)
	}
	return
}

// ---------------------------------------------------------------------------------
func PangWSController(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangWSController:", qo.Source, qo.Track)
	defer log.Println("OUT PangWSController:", err)

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}

	if s.chn.Blocked || !s.chn.isValidStreamKey(qo.Channel.Key) {
		err = fmt.Errorf("not allowed to use: %v, %s", s.chn.Blocked, qo.Channel.Key)
		log.Println(err)
		return
	}

	s.ChannelID = s.chn.ID
	s.chn.AtUsed = time.Now()

	for s.isState(Using) && s.chn.isState(Using) {
		rm := &WSMessage{}
		err = ws.ReadJSON(rm)
		if err != nil {
			log.Println(err)
			return
		}
		err = s.procControlMessage(ws, rm)
		if err != nil {
			log.Println(err)
			return
		}
		// SafeSendMessage(s.MsgChan, rm)
	}
	return
}

// ---------------------------------------------------------------------------------
func PangWSEventer(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangWSEventer:", qo.Channel.ID)
	defer log.Println("OUT PangWSEventer:", err)

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithNameRequest(qo.URL.Path, qo.Session.ReqID)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.findChannelByID(qo.Channel.ID)
	if s.chn == nil {
		err = fmt.Errorf("findChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}

	// --------------- prepare only one channel broker for eventing
	s.chn.addEventer(s.ID, ws)
	defer s.chn.deleteEventer(s.ID)

	if s.chn.isEventState(Idle) {
		go ChannelEventBroker(qo.URL.Path, s.chn)
		for i := 0; s.chn.isEventState(Idle) && i < 30; i++ {
			time.Sleep(100 * time.Millisecond)
		}
	}

	s.chn.pushEvent("evt-in", s.ID, s.Name, s.RequestID)
	defer s.chn.pushEvent("evt-out", s.ID, s.Name, s.RequestID)

	// --------------- keep the connection and check it
	for s.isState(Using) {
		time.Sleep(time.Second)
		err = ws.WriteMessage(websocket.PingMessage, []byte("eventer:ping"))
		if err != nil {
			log.Println(err)
			return
		}
		if s.chn.isEventState(Idle) {
			log.Println("no channel event broker")
			return
		}
		continue
	}
	return
}

// ---------------------------------------------------------------------------------
func ChannelEventBroker(path string, c *Channel) (err error) {
	log.Println("IN ChannelEventBroker:", c.ID)
	defer log.Println("OUT ChannelEventBroker:", c.ID)

	w := pStudio.addNewWorkerWithParams(path, c.ID, "system")
	defer pStudio.deleteWorker(w)

	c.setEventState(Using)
	defer c.setEventState(Idle)

	for w.isState(Using) {
		var evt EventMessage
		select {
		case evt = <-c.eventChan:
			log.Println(evt)
		default:
			time.Sleep(100 * time.Millisecond)
			if c.isNoEventers() || c.isEventState(Idle) {
				// channel broker is no more needed
				return
			}
			continue
		}

		// send events to all event receivers
		for _, ews := range c.Eventers {
			err = ews.WriteJSON(evt)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
	return
}

// ---------------------------------------------------------------------------------
// Bridge Puller: sub(ws) -> pub(int)
// ---------------------------------------------------------------------------------
func PangWSPuller(b *Bridge, ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangWSPuller:", qo.URL.Path, qo.Source, qo.Track)
	defer log.Println("OUT PangWSPuller:", err)

	b.State = Using
	b.AtUsed = time.Now()
	defer pStudio.afterSetBridgeIdleByID(b.ID)
	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithNameRequest(qo.URL.Path, qo.Session.ReqID)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}
	s.ChannelID = s.chn.ID
	s.chn.addPublisher(s)
	defer s.chn.deletePublisher(s)

	cntChannelsUsing := pStudio.countChannelsByState("using")
	if cntChannelsUsing > mConfig.NumPubs {
		err = fmt.Errorf("too many channels for license: %d/%d", cntChannelsUsing, mConfig.NumPubs)
		log.Println(err)
		return
	}

	s.src, s.trk, err = s.chn.addSourceTrackByLabel(qo.Source.Label, qo.Track.Label)
	if err != nil {
		log.Println(err)
		return
	}
	s.trk.Mode = qo.Track.Mode
	s.trk.Style = qo.Track.Style
	defer s.resetTrackInfo()

	s.BridgeID = b.ID
	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)
	s.chn.AtUsed = time.Now()

	s.chn.pushEvent("pull-in", s.ID, s.Name, s.BridgeID)
	defer s.chn.pushEvent("pull-out", s.ID, s.Name, s.BridgeID)

	err = s.trk.handleBuffersByPangWSAPI(ws, s, qo.Track.Mode) // API:/pang/int/pub
	return
}

// ---------------------------------------------------------------------------------
// Bridge Pusher: sub(int) -> pub(ws)
// ---------------------------------------------------------------------------------
func PangWSPusher(b *Bridge, ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangWSPusher:", qo.URL.Path, qo.Source, qo.Track)
	defer log.Println("OUT PangWSPusher:", err)

	b.State = Using
	b.AtUsed = time.Now()
	defer pStudio.afterSetBridgeIdleByID(b.ID)
	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithNameRequest(qo.URL.Path, qo.Session.ReqID)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}
	s.ChannelID = s.chn.ID
	s.chn.addSubscriber(s)
	defer s.chn.deleteSubscriber(s)

	s.src, s.trk, err = s.chn.addSourceTrackByLabel(qo.Source.Label, qo.Track.Label)
	if err != nil {
		log.Println(err)
		return
	}

	s.BridgeID = b.ID
	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)
	s.chn.AtUsed = time.Now()

	s.chn.pushEvent("push-in", s.ID, s.Name, s.BridgeID)
	defer s.chn.pushEvent("push-out", s.ID, s.Name, s.BridgeID)

	err = s.trk.handleBuffersByPangWSAPI(ws, s, qo.Track.Mode) // API:/pang/int/sub
	return
}

// ---------------------------------------------------------------------------------
// API: /pang/ws/pub, communication model : 1 -> 1
// ---------------------------------------------------------------------------------
func PangWSPublisher(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangWSPublisher:", qo.Source, qo.Track)
	defer log.Println("OUT PangWSPublisher:", err)

	runtime.LockOSThread() // Is this really effective?

	// -- check if there is only one session
	if !pStudio.checkResourceAvailable(qo) {
		err = fmt.Errorf("resource [%s/%s/%s] already used",
			qo.Channel.ID, qo.Source.Label, qo.Track.Label)
		log.Println(err)
		return
	}

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithNameRequest(qo.URL.Path, qo.Session.ReqID)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}
	s.ChannelID = s.chn.ID
	s.chn.addPublisher(s)
	defer s.chn.deletePublisher(s)

	cntChannelsUsing := pStudio.countChannelsByState("using")
	if cntChannelsUsing > mConfig.NumPubs {
		err = fmt.Errorf("too many channels for license: %d/%d", cntChannelsUsing, mConfig.NumPubs)
		log.Println(err)
		return
	}

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
	s.trk.Mode = qo.Track.Mode
	s.trk.Style = qo.Track.Style
	defer s.resetTrackInfo()

	// NOTICE: Testing for track buffer size by query option
	s.trk.setTrackBufferSize(qo.Buffer.Len)

	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)
	s.chn.AtUsed = time.Now()

	s.chn.pushEvent("pub-in", s.ID, s.Name, s.RequestID)
	defer s.chn.pushEvent("pub-out", s.ID, s.Name, s.RequestID)

	err = s.trk.handleBuffersByPangWSAPI(ws, s, qo.Track.Mode)
	return
}

// ---------------------------------------------------------------------------------
// API: /pang/ws/sub, communication model : 1 -> N
// ---------------------------------------------------------------------------------
func PangWSSubscriber(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangWSSubscriber:", qo.Source, qo.Track)
	defer log.Println("OUT PangWSSubscriber:", err)

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithNameRequest(qo.URL.Path, qo.Session.ReqID)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}
	s.ChannelID = s.chn.ID
	s.chn.addSubscriber(s)
	defer s.chn.deleteSubscriber(s)

	cntSessionsUsing := pStudio.countSessionsByState("using")
	if cntSessionsUsing > mConfig.NumSubs {
		err = fmt.Errorf("too many sessions for license: %d/%d", cntSessionsUsing, mConfig.NumSubs)
		log.Println(err)
		return
	}

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
	s.chn.AtUsed = time.Now()

	s.chn.pushEvent("sub-in", s.ID, s.Name, s.RequestID)
	defer s.chn.pushEvent("sub-out", s.ID, s.Name, s.RequestID)

	err = s.trk.handleBuffersByPangWSAPI(ws, s, qo.Track.Mode)
	return
}

// ---------------------------------------------------------------------------------
// API: /pang/ws/meb, communication model : M <-> N (Medusa mode)
// ---------------------------------------------------------------------------------
func PangWSMedusa(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangWSMedusa:", qo.Source, qo.Track)
	defer log.Println("OUT PangWSMedusa:", err)

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}

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

	// -- meb method = bundle buffers, multi pubs/subs
	s.trk.Mode = "bundle" // bi-directional communication
	s.trk.Style = "multi" // multi pubs/subs

	s.ChannelID = s.chn.ID
	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)
	s.chn.AtUsed = time.Now()

	s.chn.pushEvent("meb-in", s.ID, s.Name, s.RequestID)
	defer s.chn.pushEvent("meb-out", s.ID, s.Name, s.RequestID)

	err = s.trk.handleBuffersByPangWSAPI(ws, s, qo.Track.Mode)
	return
}

// ---------------------------------------------------------------------------------
// API: /pang/ws/zeb, communication model : N -> (M) -> N (Zebra mode)
// ---------------------------------------------------------------------------------
func PangWSZebber(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangWSZebber:", qo.Source, qo.Track)
	defer log.Println("OUT PangWSZebber:", err)

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}

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

	s.ChannelID = s.chn.ID
	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)
	s.chn.AtUsed = time.Now()

	// -- zeb method = single buffer, multi pubs
	s.trk.Mode = "multi"  // multi buffer
	s.trk.Style = "multi" // multi pubs

	s.chn.pushEvent("zeb-in", s.ID, s.Name, s.RequestID)
	defer s.chn.pushEvent("zeb-out", s.ID, s.Name, s.RequestID)

	err = s.trk.handleBuffersByPangWSAPI(ws, s, qo.Track.Mode)
	return
}

// ---------------------------------------------------------------------------------
func (trk *Track) handleBuffersByPangWSAPI(ws *websocket.Conn, s *Session, mode string) (err error) {
	log.Println("i.handleBuffersByPangWSAPI:", s.Name)
	defer log.Println("o.handleBuffersByPangWSAPI:", err)

	switch s.Name {
	case "/pang/ws/pub": // publisher type
		rbuf := trk.Rings[BUFFER_NUM_FORE] // [0]: foreward direction
		sbuf := trk.Rings[BUFFER_NUM_BACK] // [1]: backward direction
		if mode == "bundle" {              // bi-directional
			go sbuf.sendTrackBufferInWSMessage(ws, s, false) // sender routine
		}
		// rbuf.setBufferSizeLen(10) // for testing
		err = rbuf.recvTrackBufferInWSMessage(ws, s, false) // receiver routine
	case "/pang/ws/sub": // subscriber type
		rbuf := trk.Rings[BUFFER_NUM_BACK] // [1]: backward direction
		sbuf := trk.Rings[BUFFER_NUM_FORE] // [0]: forward direction
		if mode == "bundle" {              // bi-directional
			go rbuf.recvTrackBufferInWSMessage(ws, s, true) // receiver routine
		}
		err = sbuf.sendTrackBufferInWSMessage(ws, s, true) // sender routine
	case "/pang/ws/meb": // broadcast type apps: text, voice chat, data hub
		rbuf := trk.Rings[BUFFER_NUM_FORE]                  // [0]: both direction,
		sbuf := trk.Rings[BUFFER_NUM_FORE]                  // [0]: single buffer
		go rbuf.recvTrackBufferInWSMessage(ws, s, true)     // from multi pubs
		err = sbuf.sendTrackBufferInWSMessage(ws, s, false) // to multi subs
	default:
		err = fmt.Errorf("not support Pang WS API: %s", s.Name)
	}
	return
}

// ---------------------------------------------------------------------------------
// sendTrackBufferInWSMessage(timeout) : sender routine for the buffer
// ---------------------------------------------------------------------------------
func (b *Buffer) sendTrackBufferInWSMessage(ws *websocket.Conn, s *Session, fout bool) (err error) {
	log.Println("i.sendTrackBufferInWSMessage:", s.trk.Label)
	defer log.Println("o.sendTrackBufferInWSMessage:", err)

	defer s.setState(Idle)

	// send the mime information for the track
	if s.chn.isState(Using) && s.trk.Mime != "" {
		err = ws.WriteMessage(websocket.TextMessage, []byte(s.trk.Mime))
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(s.trk.Mime)
	}

	lpos := b.PosWrite
	etime := time.Now().Add(s.TimeOver)

	// extend the websocket timeout by ping message
	ws.SetPingHandler(func(msg string) (err error) {
		etime = time.Now().Add(s.TimeOver)
		log.Println("sendBuffer ping:", msg)
		return
	})

	// send slots in the buffer while the session and channel are using
	for s.isState(Using) && s.chn.isState(Using) {
		if lpos == b.PosWrite {
			if time.Now().After(etime) {
				if fout { // if the timeout is set, then return
					log.Println("timeout:", s.TimeOver, s.TimeUnit)
					return
				}
			}
			time.Sleep(s.TimeUnit)
			continue
		}
		etime = time.Now().Add(s.TimeOver)

		bs := b.readSlotByPos(lpos)

		if bs.Head.(string) != s.ID { // skip the self message
			err = ws.WriteMessage(bs.FrameType, bs.Data)
			if err != nil {
				log.Println(err)
				return
			}

			s.OutBytes += bs.Length
			s.trk.OutBytes += bs.Length
			s.chn.OutBytes += bs.Length
		}

		lpos = b.setReadPos(lpos)
	}
	return
}

// ---------------------------------------------------------------------------------
// recvTrackBufferInWSMessage(locking) : receiver routine for the buffer
// ---------------------------------------------------------------------------------
func (b *Buffer) recvTrackBufferInWSMessage(ws *websocket.Conn, s *Session, flock bool) (err error) {
	log.Println("i.recvTrackBufferInWSMessage:", s.trk.Label)
	defer log.Println("o.recvTrackBufferInWSMessage:", err)

	defer s.setState(Idle)

	// extend the websocket timeout
	ws.SetPingHandler(func(msg string) (err error) {
		s.setWebSocketTimeout(ws)
		log.Println("recvBuffer ping:", msg)
		// ws.WriteMessage(websocket.PongMessage, []byte("pong"))
		return
	})

	for s.isState(Using) && s.chn.isState(Using) {
		s.setWebSocketTimeout(ws)
		bs := Slot{Head: s.ID, FrameType: websocket.BinaryMessage, Mark: RSSP_MARK_RBIN}
		bs.FrameType, bs.Data, err = ws.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		if bs.FrameType == websocket.TextMessage {
			bs.Mark = RSSP_MARK_RTXT
			if IsExtTextMessage(bs.Data) {
				err = ProcExtTextMessage(s, s.trk, bs.Data)
				if err != nil {
					log.Println("ProcExtTextMessage:", err)
				}
			} else {
				s.trk.Mime = string(bs.Data)
				log.Println(s.Name, s.trk.Label, s.trk.Mime)
			}
		}

		bs.getLengthTime()
		b.writeSlot(bs, flock)

		s.InBytes += bs.Length
		s.trk.InBytes += bs.Length
		s.chn.InBytes += bs.Length
	}
	return
}

// ---------------------------------------------------------------------------------
func IsExtTextMessage(data []byte) (ret bool) {
	if len(data) < 8 {
		return false
	}
	head := string(data[:4])
	if head == "REXT" {
		return true // Extended Text Message in RSSP format
	}
	return false
}

// ---------------------------------------------------------------------------------
func ProcExtTextMessage(s *Session, trk *Track, data []byte) (err error) {
	if len(data) < 12 {
		return fmt.Errorf("invalid ext msg length: %d", len(data))
	}
	xhead := string(data[4:8]) // Head of Ext Message
	xbody := string(data[8:])  // Body of Ext Message
	switch xhead {
	case "MIME": // Text MIME Message (new style)
		trk.Mime = xbody
		log.Println("MIME:", trk.Mime)
	case "CARD": // Text Agent Card Message
		trk.Cards[s.ID] = xbody
		log.Println("CARD:", trk.Cards[s.ID])
	case "XCMD": // Text Command Message
	case "XACK": // Text Acknowledgement Message
	case "XERR": // Text Error Message
	default:
		err = fmt.Errorf("unknown ext msg head: %s", xhead)
	}
	return
}

// ---------------------------------------------------------------------------------
// API: /pang/ws/p2p, direct communication model : 1 <-> (X) <-> 1
// ---------------------------------------------------------------------------------
func PangWSPeerDirect(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangWSPeerDirect:", qo.Source, qo.Track)
	defer log.Println("OUT PangWSPeerDirect:", err)

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}

	s.src, s.trk, err = s.chn.addSourceTrackByLabel(qo.Source.Label, qo.Track.Label)
	if err != nil {
		log.Println(err)
		return
	}

	s.trk.Mode = "bundle" // bi-directional communication in default
	defer s.trk.resetTrackInfo()

	s.ChannelID = s.chn.ID
	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)
	s.chn.AtUsed = time.Now()

	s.chn.pushEvent("p2p-in", s.ID, s.Name, s.RequestID)
	defer s.chn.pushEvent("p2p-out", s.ID, s.Name, s.RequestID)

	// manage the peers in the same track
	err = s.registerPeer(ws)
	if err != nil {
		log.Println(err)
		return
	}
	defer s.unregisterPeer()

	pws, err := s.setupPeerConnection()
	if err != nil {
		log.Println(err)
		return
	}

	for s.isState(Using) {
		mt, msg, err := pws.ReadMessage()
		if err != nil {
			log.Println("recv", err)
			break
		}

		err = ws.WriteMessage(mt, msg)
		if err != nil {
			log.Println("send:", err)
			break
		}

		// --- Below is used when both peers not disconnected on an error
		// mt, msg, err := ws.ReadMessage()
		// if err != nil {
		// 	log.Println("recv", err)
		// 	break
		// }
		//
		// pws := s.getPeerWebSocket()
		// if pws == nil {
		// 	log.Println("peer ws is not ready")
		// 	continue
		// }
		//
		// err = pws.WriteMessage(mt, msg)
		// if err != nil {
		// 	log.Println(err)
		// 	continue
		// }
	}
	return
}

// ---------------------------------------------------------------------------------
func (s *Session) tossMessageToBuffer(b *Buffer, mt int, msg []byte) (err error) {
	bs := Slot{Head: s.ID, FrameType: mt, Mark: RSSP_MARK_RBIN}
	bs.Data = msg

	if bs.FrameType == websocket.TextMessage {
		bs.Mark = RSSP_MARK_RTXT
		b.Mime = string(bs.Data)
		log.Println(s.Name, s.trk.Label, s.trk.Mime, b.Mime)
	}

	bs.getLengthTime()
	b.writeSlot(bs, false)

	s.InBytes += bs.Length
	s.trk.InBytes += bs.Length
	s.chn.InBytes += bs.Length
	return
}

// ---------------------------------------------------------------------------------
func (s *Session) setupPeerConnection() (pws *websocket.Conn, err error) {
	// log.Println("i.setupPeerConnection:", s.ID)

	err = s.waitPeerConnection()
	if err != nil {
		log.Println(err)
		return
	}

	pws = s.getPeerWebSocket()
	if pws == nil {
		err = fmt.Errorf("peer ws is invalid")
		return
	}
	return
}

// ---------------------------------------------------------------------------------
func (s *Session) registerPeer(ws *websocket.Conn) (err error) {
	if len(s.chn.Peers) < 2 {
		s.addPeerInfo(ws)
	} else {
		err = fmt.Errorf("too many ws peers (%d) in the channel", len(s.chn.Peers))
	}
	return
}

// ---------------------------------------------------------------------------------
func (s *Session) unregisterPeer() {
	defer s.deletePeerInfo(s.ID)
}

// ---------------------------------------------------------------------------------
func (s *Session) addPeerInfo(ws *websocket.Conn) {
	s.chn.Lock()
	defer s.chn.Unlock()
	s.ws = ws
	s.chn.Peers[s.ID] = s
}

// ---------------------------------------------------------------------------------
func (s *Session) deletePeerInfo(id string) {
	s.chn.Lock()
	defer s.chn.Unlock()
	delete(s.chn.Peers, id)
}

// ---------------------------------------------------------------------------------
func (s *Session) waitPeerConnection() (err error) {
	// to avoid the zero value of time.Duration
	if s.TimeUnit == 0 {
		s.TimeUnit = time.Millisecond
	}

	for i := 0; s.isState(Using); i++ {
		if len(s.chn.Peers) == 2 {
			return
		}
		if i < (int)(s.TimeOver/s.TimeUnit) { // possible to divide by zero
			time.Sleep(s.TimeUnit)
			continue
		}
		err = fmt.Errorf("peer ws wait timeout: %v, %v", s.TimeOver, s.TimeUnit)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
func (s *Session) getPeerWebSocket() (pws *websocket.Conn) {
	s.chn.Lock()
	defer s.chn.Unlock()
	for k, v := range s.chn.Peers {
		if k != s.ID { // except myself
			if v.SourceID == s.SourceID && v.TrackID == s.TrackID {
				pws = v.ws
				log.Println("peer ws session:", k)
				break
			}
		}
	}
	return
}

//=================================================================================
