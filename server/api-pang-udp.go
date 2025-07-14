// =================================================================================
// Filename: api-pang-udp.go
// Function: pang websocket API for UDP based streaming
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2022 - 2024
// =================================================================================
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
const (
	MAX_SLOT_SIZE     = 1024 * 1024
	MAX_UDP_PKT_SIZE  = 65536
	RSSP_UDP_MTU_SIZE = 1500 // max udp packet size cf) 1500 (ethernet mtu)
	RSSP_UDP_MSG_SIZE = 1472 // // max udp message size cf) 64KB (65535 = 20 + 8 + 65507), < 1400 (low bw network)
)

// ---------------------------------------------------------------------------------
func PangUDPPublisher(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangUDPPublisher:", qo.Source, qo.Track, qo.Stream.Addr)
	defer log.Println("OUT PangUDPPublisher:", err)

	if !pStudio.checkResourceAvailable(qo) {
		err = fmt.Errorf("resource [%s/%s/%s] already used",
			qo.Channel.ID, qo.Source.Label, qo.Track.Label)
		log.Println(err)
		return
	}

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("setChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}
	s.chn.AtUsed = time.Now()
	s.ChannelID = s.chn.ID

	if s.chn.Blocked || !s.chn.isValidStreamKey(qo.Channel.Key) {
		err = fmt.Errorf("not allowed to use: %v, %s", s.chn.Blocked, qo.Channel.Key)
		log.Println(err)
		return
	}

	_, s.trk, err = s.chn.addSourceTrackByLabel(qo.Source.Label, qo.Track.Label)
	if err != nil {
		log.Println(err)
		return
	}
	s.trk.Mode = qo.Track.Mode
	s.trk.Style = qo.Track.Style

	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)

	//-------------------------------------------------------
	for s.isState(Using) {
		rm := &WSMessage{}
		err = ws.ReadJSON(rm)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println(rm)

		sm := &WSMessage{}
		switch rm.Type {
		case "ping":
			sm.Type = "pong"
		case "offer":
			err = json.Unmarshal([]byte(rm.Data), &qo.Stream)
			if err != nil {
				log.Println(err)
				return
			}
			log.Println(qo.Stream.Addr)
			s.trk.Mime = qo.Stream.Mime

			udp, uaddr, err := openUDPRecvPort("udp", ":0")
			if err != nil {
				log.Println(err)
				return err
			}
			defer closeUDPPort(udp)

			go s.trk.handleBuffersByPangUDPAPI(udp, s)

			sm.Type = "answer"
			qo.Stream.Addr = uaddr
			data, _ := json.Marshal(qo.Stream)
			sm.Data = string(data)
		default:
			log.Println("unknown message type:", rm.Type)
			continue
		}

		err = ws.WriteJSON(sm)
		if err != nil {
			log.Println(err)
			return
		}
	}
	return
}

// ---------------------------------------------------------------------------------
func PangUDPSubscriber(ws *websocket.Conn, qo QueryOption) (err error) {
	log.Println("IN PangUDPSubscriber:", qo.Source, qo.Track)
	defer log.Println("OUT PangUDPSubscriber:", err)

	defer pStudio.afterSetChannelIdleByID(qo.Channel.ID)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	s.chn = pStudio.setChannelByIDState(qo.Channel.ID, Using)
	if s.chn == nil {
		err = fmt.Errorf("getChannel(%s) is invalid", qo.Channel.ID)
		log.Println(err)
		return
	}
	s.chn.AtUsed = time.Now()
	s.ChannelID = s.chn.ID

	if s.chn.Blocked || !s.chn.isValidStreamKey(qo.Channel.Key) {
		err = fmt.Errorf("not allowed to use: %v, %s", s.chn.Blocked, qo.Channel.Key)
		log.Println(err)
		return
	}

	_, s.trk, err = s.chn.addSourceTrackByLabel(qo.Source.Label, qo.Track.Label)
	if err != nil {
		log.Println(err)
		return
	}

	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)

	//-------------------------------------------------------
	for s.isState(Using) {
		rm := &WSMessage{}
		err = ws.ReadJSON(rm)
		if err != nil {
			log.Println(err)
			break
		}
		log.Println(rm)

		sm := &WSMessage{}
		switch rm.Type {
		case "ping":
			sm.Type = "pong"
		case "offer":
			err = json.Unmarshal([]byte(rm.Data), &qo.Stream)
			if err != nil {
				log.Println(err)
				return
			}

			rname := qo.Stream.Addr.Host + ":" + qo.Stream.Addr.Port
			udp, uaddr, err := openUDPSendPort("udp", ":0", rname)
			if err != nil {
				log.Println(err)
				return err
			}
			defer closeUDPPort(udp)

			go s.trk.handleBuffersByPangUDPAPI(udp, s)

			sm.Type = "answer"
			qo.Stream.Mime = s.trk.Mime
			qo.Stream.Addr = uaddr
			data, _ := json.Marshal(qo.Stream)
			sm.Data = string(data)
			log.Println(sm)
		default:
			log.Println("unknown message type:", rm.Type)
			continue
		}

		err = ws.WriteJSON(sm)
		if err != nil {
			log.Println(err)
			return
		}
	}
	return
}

// ---------------------------------------------------------------------------------
func openUDPSendPort(network, lname, rname string) (udp *net.UDPConn, addr Addr, err error) {
	laddr, err := net.ResolveUDPAddr(network, lname)
	if err != nil {
		log.Println(err)
		return
	}
	raddr, err := net.ResolveUDPAddr(network, rname)
	if err != nil {
		log.Println(err)
		return
	}
	udp, err = net.DialUDP(network, laddr, raddr) // checking the ready for receiving
	if err != nil {
		log.Println(err)
		return
	}
	addr.Network = network
	addr.Host, addr.Port, err = net.SplitHostPort(udp.RemoteAddr().String())
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func openUDPRecvPort(network, lname string) (udp *net.UDPConn, addr Addr, err error) {
	laddr, err := net.ResolveUDPAddr(network, lname)
	if err != nil {
		log.Println(err)
		return
	}
	udp, err = net.ListenUDP(network, laddr)
	if err != nil {
		log.Println(err)
		return
	}
	addr.Network = network
	addr.Host, addr.Port, err = net.SplitHostPort(udp.LocalAddr().String())
	if err != nil {
		log.Println(err)
		return
	}
	return
}

func closeUDPPort(udp *net.UDPConn) (err error) {
	if udp != nil {
		udp.Close()
	}
	return
}

// ---------------------------------------------------------------------------------
func (trk *Track) handleBuffersByPangUDPAPI(udp *net.UDPConn, s *Session) (err error) {
	log.Println("i.handleBuffersByPangUDPAPI:", s.Name)
	defer log.Println("o.handleBuffersByPangUDPAPI:", err)

	switch s.Name {
	case "/pang/udp/pub":
		rbuf := trk.Rings[BUFFER_NUM_FORE] // 0: foreward direction
		err = rbuf.recvTrackBufferInUDPMessage(udp, s, true)
	case "/pang/udp/sub":
		sbuf := trk.Rings[BUFFER_NUM_FORE] // 0: forward direction
		err = sbuf.sendTrackBufferInUDPMessage(udp, s, true)
	default:
		err = fmt.Errorf("not support pang UDP API: %s", s.Name)
	}
	return
}

// ---------------------------------------------------------------------------------
func (b *Buffer) recvTrackBufferInUDPMessage(udp *net.UDPConn, s *Session, flock bool) (err error) {
	log.Println("i.recvTrackBufferInUDPMessage:")
	defer log.Println("o.recvTrackBufferInUDPMessage:", err)

	buf := make([]byte, MAX_SLOT_SIZE)

	for s.isState(Using) && s.chn.isState(Using) {
		udp.SetReadDeadline(time.Now().Add(10 * time.Second))
		bs := Slot{Head: s.ID}
		n, addr, err := udp.ReadFrom(buf)
		if err != nil {
			log.Println(err, n, addr)
			break
		}
		bs.FrameType = websocket.BinaryMessage
		bs.Data = buf[:n]
		bs.getLengthTime()
		// fmt.Println(addr, n)

		b.writeSlot(bs, flock)

		s.InBytes += bs.Length
		s.trk.InBytes += bs.Length
		s.chn.InBytes += bs.Length
	}
	return
}

func (b *Buffer) sendTrackBufferInUDPMessage(udp *net.UDPConn, s *Session, fout bool) (err error) {
	log.Println("i.sendTrackBufferInUDPMessage:")
	defer log.Println("o.sendTrackBufferInUDPMessage:", err)

	lpos := b.PosWrite
	etime := time.Now().Add(s.TimeOver)

	for s.isState(Using) && s.chn.isState(Using) {
		if lpos == b.PosWrite {
			if time.Now().After(etime) {
				if fout {
					log.Println("timeout:", s.TimeOver, s.TimeUnit)
					return
				}
			}
			time.Sleep(s.TimeUnit)
			continue
		}
		etime = time.Now().Add(s.TimeOver)

		bs := b.readSlotByPos(lpos)

		if bs.Head.(string) != s.ID { // ignore its self messages
			n, err := udp.Write(bs.Data)
			if err != nil {
				log.Println(err, n)
				break
			}
			s.OutBytes += bs.Length
			s.trk.OutBytes += bs.Length
			s.chn.OutBytes += bs.Length
		}

		lpos = (lpos + 1) % b.SizeLen // CAUTION: lpos != trk.RPos
	}
	return
}

// ---------------------------------------------------------------------------------
// RSSP protocol part over UDP
// ---------------------------------------------------------------------------------
func UDPSendMessageRetry(conn *net.UDPConn, addr *net.UDPAddr, timeover time.Duration, suffix string, data []byte, retry int) (err error) {
	// log.Println("i.UDPSendMessageRetry:", timeover, n)

	for i := 0; i < retry; i++ {
		err = UDPSendMessageAck(conn, addr, timeover, suffix, data)
		if err != nil {
			log.Println(err)
			continue
		}
		break
	}
	return
}

// ---------------------------------------------------------------------------------
func UDPSendMessageAck(conn *net.UDPConn, addr *net.UDPAddr, timeover time.Duration, suffix string, data []byte) (err error) {
	// log.Println("i.sendUDPMessageAck:", addr, timeover)

	err = UDPSendMessage(conn, addr, timeover, suffix, data)
	if err != nil {
		log.Println(err)
		return
	}
	sendLength := len(data)

	suffix, data, _, err = UDPRecvMessage(conn, timeover)
	if err != nil {
		log.Println(err, suffix)
		return
	}
	recvLength := len(data)

	if sendLength != recvLength {
		err = fmt.Errorf("send(%d) != recv(%d)", sendLength, recvLength)
		log.Println(err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
func UDPRecvMessageAck(conn *net.UDPConn, timeover time.Duration) (suffix string, data []byte, addr *net.UDPAddr, err error) {
	// log.Println("UDPRecvMessageAck:", timeover)

	suffix, data, addr, err = UDPRecvMessage(conn, timeover)
	if err != nil {
		log.Println(err, suffix)
		return
	}

	err = UDPSendMessage(conn, addr, timeover, suffix, data)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
// RSSP UDP Message Format (suffix style): [data|Length|4CC]
// ---------------------------------------------------------------------------------
func UDPSendMessage(conn *net.UDPConn, addr *net.UDPAddr, timeover time.Duration, suffix string, data []byte) (err error) {
	// log.Println("i.UDPSendMessage:")

	conn.SetWriteDeadline(time.Now().Add(timeover))

	dataLength := uint32(len(data))
	message := make([]byte, len(data)+4+4)

	copy(message[:dataLength], []byte(data))
	binary.BigEndian.PutUint32(message[dataLength:dataLength+4], dataLength)
	copy(message[dataLength+4:dataLength+8], []byte(suffix))

	for len(message) > RSSP_UDP_MSG_SIZE {
		n, err := conn.Write(message[:RSSP_UDP_MSG_SIZE])
		if err != nil {
			log.Println("1. UDP Write", n, err)
			return err
		}
		message = message[RSSP_UDP_MSG_SIZE:]
	}

	if len(message) > 0 {
		n, err := conn.Write(message)
		if err != nil {
			log.Println("2. UDP Write", n, err)
			return err
		}
	}
	return
}

// ---------------------------------------------------------------------------------
func UDPRecvMessage(conn *net.UDPConn, timeover time.Duration) (suffix string, data []byte, addr *net.UDPAddr, err error) {
	// log.Println("UDPRecvMessage:", timeover)

	conn.SetReadDeadline(time.Now().Add(timeover))

	buffer := make([]byte, RSSP_MAX_DATA_SIZE)
	for {
		n, caddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Println("UDP Read", err)
			return "", nil, addr, err
		}

		addr = caddr
		data = append(data, buffer[:n]...)

		if len(data) > 8 {
			suffix = string(data[len(data)-RSSP_MARK_SIZE:])
			if suffix == RSSP_MARK_RBIN || suffix == RSSP_MARK_RTXT {
				break
			}
		}
	}

	lengthBytes := data[(len(data) - 8) : len(data)-4]
	dataLength := binary.BigEndian.Uint32(lengthBytes) // default is big endian

	data = data[:(len(data) - 8)]
	if uint32(len(data)) != dataLength {
		log.Println(len(data), dataLength, suffix)
	}
	return
}

//=================================================================================
