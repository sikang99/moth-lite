// =================================================================================
// Filename: api-pang-tcp.go
// Function: pang tcp API for TCP based streaming
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2022 - 2024
// =================================================================================
package main

import (
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
// API: /pang/tcp/tst, for testing
// ---------------------------------------------------------------------------------
func PangTCPTester(conn net.Conn, qo QueryOption) (err error) {
	log.Println("IN PangTCPTester:", qo.Channel, qo.Session.Unit)
	defer log.Println("OUT PangTCPTester:", err)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	log.Println("TBD:Tester")

	return
}

// ---------------------------------------------------------------------------------
// API: /pang/tcp/p2p, for peering two parts directly
// ---------------------------------------------------------------------------------
func PangTCPPeering(conn net.Conn, qo QueryOption) (err error) {
	log.Println("IN PangTCPPeering:", qo.Channel, qo.Session.Unit)
	defer log.Println("OUT PangTCPPeering:", err)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)

	var pnh *Punch = nil

	// log.Println(qo.Stream.Role)
	switch qo.Stream.Role {
	case "/pang/tcp/pub":
		lph := NewPunchPointer()
		lph.Name = fmt.Sprintf("/%s/%s/%s", qo.Channel.ID, qo.Source.Label, qo.Track.Label)
		lph.ChannelID = qo.Channel.ID
		lph.SourceID = qo.Source.Label
		lph.TrackID = qo.Track.Label
		lph.Addr = conn.RemoteAddr().String()
		lph.Role = qo.Stream.Role
		lph.ResourceID = lph.Name

		pnh = pStudio.findPunchByName(lph.Name)
		if pnh == nil {
			pnh = pStudio.addPunch(lph) // add new punch if not exist
		} else {
			pnh.Addr = conn.RemoteAddr().String() // update punch address if exist
		}
		pnh.setState(Idle)
	case "/pang/tcp/sub":
		name := fmt.Sprintf("/%s/%s/%s", qo.Channel.ID, qo.Source.Label, qo.Track.Label)
		pnh = pStudio.findPunchByName(name)
		if pnh == nil {
			err = fmt.Errorf("not found punch by name: %s", name)
			log.Println(err)
			return
		}
		pnh.setState(Using)
	case "/pang/tcp/p2p/pub", "/pang/tcp/p2p/sub":
		name := fmt.Sprintf("/%s/%s/%s", qo.Channel.ID, qo.Source.Label, qo.Track.Label)
		pnh = pStudio.findPunchByName(name)
		if pnh == nil {
			err = fmt.Errorf("not found punch by name: %s", name)
			log.Println(err)
			return
		}
		pnh.SessionID = s.ID
		s.Name = qo.Stream.Role
		for { // receive heartbeat message from the peer
			prefix, _, err := TCPRecvMessage(conn, s.TimeOver) // need to be control message for just checking
			if err != nil {
				log.Println(prefix, err)
				break
			}
			pnh.AtExpired = time.Now().Add(s.TimeOver)
		}
		if qo.Stream.Role == "/pang/tcp/p2p/pub" {
			pStudio.deletePunch(pnh)
		}
		return
	}

	if pnh == nil {
		err = fmt.Errorf("not found a proper p2p punch")
		log.Println(err)
		return
	}
	pnh.AtUsed = time.Now()

	data, err := json.Marshal(pnh)
	if err != nil {
		log.Println(err)
		return err
	}

	n, err := TCPSendMessage(conn, s.TimeOver, RSSP_MARK_RTXT, []byte(data))
	if err != nil {
		log.Println(n, err)
		return err
	}
	time.Sleep(time.Second)
	return
}

// ---------------------------------------------------------------------------------
// API: /pang/tcp/eco, communication model : 1 <-> 0
// ---------------------------------------------------------------------------------
func PangTCPReflector(conn net.Conn, qo QueryOption) (err error) {
	log.Println("IN PangTCPReflector:", qo.Channel, qo.Session.Unit)
	defer log.Println("OUT PangTCPReflector:", err)

	s := pStudio.addNewSessionWithName(qo.URL.Path)
	defer pStudio.deleteSessionWithClose(s)

	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)

	for s.isState(Using) {
		prefix, data, err := TCPRecvMessage(conn, 3*time.Second)
		if err != nil {
			log.Println(prefix, err)
			break
		}
		s.InBytes += len(data)

		// simulate the buffering delay
		time.Sleep(s.TimeUnit)

		n, err := TCPSendMessage(conn, 3*time.Second, prefix, data)
		if err != nil {
			log.Println(n, err)
			break
		}
		s.OutBytes += len(data)
	}
	return
}

// ---------------------------------------------------------------------------------
// API: /pang/tcp/pub, communication model : 1 (Publisher) -> 1 (Server)
// ---------------------------------------------------------------------------------
func PangTCPPublisher(conn net.Conn, qo QueryOption) (err error) {
	log.Println("IN PangTCPPublisher:", qo.Source, qo.Track)
	defer log.Println("OUT PangTCPPublisher:", err)

	defer conn.Close()

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

	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)

	s.chn.pushEvent("pub-in", s.ID, s.Name, s.RequestID)
	defer s.chn.pushEvent("pub-out", s.ID, s.Name, s.RequestID)

	err = s.trk.handleBuffersByPangTCPAPI(conn, s, qo.Track.Mode)
	return
}

// ---------------------------------------------------------------------------------
// API: /pang/tcp/sub, communication model : 1 (Server) -> N (Subscribers)
// ---------------------------------------------------------------------------------
func PangTCPSubscriber(conn net.Conn, qo QueryOption) (err error) {
	log.Println("IN PangTCPSubscriber:", qo.Source, qo.Track)
	defer log.Println("OUT PangTCPSubscriber:", err)

	defer conn.Close()

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

	err = s.trk.handleBuffersByPangTCPAPI(conn, s, qo.Track.Mode)
	return
}

// ---------------------------------------------------------------------------------
// API: /pang/tcp/meb, communication model : M <-> N (Medusa mode)
// ---------------------------------------------------------------------------------
func PangTCPMedusa(conn net.Conn, qo QueryOption) (err error) {
	log.Println("IN PangTCPMedusa:", qo.Source, qo.Track)
	defer log.Println("OUT PangTCPMedusa:", err)

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

	// -- meb method = single buffer, multi pubs
	s.trk.Mode = "bundle" // bi-directional communication
	s.trk.Style = "multi" // multi pubs

	s.ChannelID = s.chn.ID
	s.SourceID = qo.Source.Label
	s.TrackID = qo.Track.Label
	s.setTimeoutInUnit(qo.Session.Timeout, qo.Session.Unit)
	s.chn.AtUsed = time.Now()

	s.chn.pushEvent("meb-in", s.ID, s.Name, s.RequestID)
	defer s.chn.pushEvent("meb-out", s.ID, s.Name, s.RequestID)

	err = s.trk.handleBuffersByPangTCPAPI(conn, s, qo.Track.Mode)
	return
}

// ---------------------------------------------------------------------------------
func (trk *Track) handleBuffersByPangTCPAPI(conn net.Conn, s *Session, mode string) (err error) {
	log.Println("i.handleBuffersByPangTCPAPI:", s.Name)
	defer log.Println("o.handleBuffersByPangTCPAPI:", err)

	switch s.Name {
	case "/pang/tcp/pub": // publisher type
		rbuf := trk.Rings[BUFFER_NUM_FORE] // [0]: foreward direction
		sbuf := trk.Rings[BUFFER_NUM_BACK] // [1]: backward direction
		if mode == "bundle" {              // bi-directional
			go sbuf.sendTrackBufferInTCPMessage(conn, s, false) // sender routine
		}
		// rbuf.setBufferSizeLen(10) // for testing
		err = rbuf.recvTrackBufferInTCPMessage(conn, s, false) // receiver routine
	case "/pang/tcp/sub": // subscriber type
		rbuf := trk.Rings[BUFFER_NUM_BACK] // [1]: backward direction
		sbuf := trk.Rings[BUFFER_NUM_FORE] // [0]: forward direction
		if mode == "bundle" {              // bi-directional
			go rbuf.recvTrackBufferInTCPMessage(conn, s, true) // receiver routine
		}
		err = sbuf.sendTrackBufferInTCPMessage(conn, s, true) // sender routine
	case "/pang/tcp/meb": // broadcast type apps: text, voice chat, data hub
		rbuf := trk.Rings[BUFFER_NUM_FORE]                     // [0]: both direction,
		sbuf := trk.Rings[BUFFER_NUM_FORE]                     // [0]: single buffer
		go rbuf.recvTrackBufferInTCPMessage(conn, s, true)     // from multi pubs
		err = sbuf.sendTrackBufferInTCPMessage(conn, s, false) // to multi subs
	case "/pang/tcp/d2m": // p2p monitoring mode
		if mode == "bundle" { // backward = subscriber
			sbuf := trk.Rings[BUFFER_NUM_BACK]
			err = sbuf.sendTrackBufferInTCPMessage(conn, s, true) // timeout is set
		} else { // == "single", forward = publisher
			sbuf := trk.Rings[BUFFER_NUM_FORE]
			err = sbuf.sendTrackBufferInTCPMessage(conn, s, true)
		}
	case "/pang/tcp/p2p": // do nothing because of no buffering
		err = fmt.Errorf("not used Pang TCP API: %s", s.Name)
	default:
		err = fmt.Errorf("not support Pang TCP API: %s", s.Name)
	}
	return
}

// ---------------------------------------------------------------------------------
// sendTrackBufferInTCPMessage(timeout) : sender routine for the buffer
// ---------------------------------------------------------------------------------
func (b *Buffer) sendTrackBufferInTCPMessage(conn net.Conn, s *Session, fout bool) (err error) {
	log.Println("i.sendTrackBufferInTCPMessage:", s.trk.Label)
	defer log.Println("o.sendTrackBufferInTCPMessage:", err)

	defer s.setState(Idle)

	// send the mime information for the track
	if s.chn.isState(Using) && s.trk.Mime != "" {
		_, err = TCPSendMessage(conn, s.TimeOver, RSSP_MARK_RTXT, []byte(s.trk.Mime))
		if err != nil {
			log.Println(err)
			return
		}
		log.Println(s.trk.Mime)
	}

	lpos := b.PosWrite
	etime := time.Now().Add(s.TimeOver)

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
			_, err = TCPSendMessage(conn, s.TimeOver, bs.Mark, bs.Data)
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
// recvTrackBufferInTCPMessage(locking) : receiver routine for the buffer
// ---------------------------------------------------------------------------------
func (b *Buffer) recvTrackBufferInTCPMessage(conn net.Conn, s *Session, flock bool) (err error) {
	log.Println("i.recvTrackBufferInTCPMessage:", s.trk.Label)
	defer log.Println("o.recvTrackBufferInTCPMessage:", err)

	defer s.setState(Idle)

	for s.isState(Using) && s.chn.isState(Using) {
		bs := Slot{Head: s.ID, FrameType: websocket.BinaryMessage, Mark: RSSP_MARK_RBIN}
		bs.Mark, bs.Data, err = TCPRecvMessage(conn, s.TimeOver)
		if err != nil {
			log.Println(err)
			return
		}

		if bs.Mark == RSSP_MARK_RTXT {
			bs.FrameType = websocket.TextMessage
			s.trk.Mime = string(bs.Data)
			log.Println(s.Name, s.trk.Label, s.trk.Mime)
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
func TCPSendMessage(conn net.Conn, timeover time.Duration, prefix string, data []byte) (n int, err error) {
	// log.Println("TCPSendMessage:", timeover, suffix)

	conn.SetWriteDeadline(time.Now().Add(timeover))

	dataLength := uint32(len(data))
	message := make([]byte, 4+4+len(data)) // 8 byte header + data

	copy(message[:4], []byte(prefix))
	binary.BigEndian.PutUint32(message[4:8], dataLength)
	copy(message[8:], []byte(data))

	n, err = conn.Write(message)
	if err != nil {
		log.Println(n, err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
func TCPRecvMessage(conn net.Conn, timeover time.Duration) (prefix string, data []byte, err error) {
	// log.Println("TCPRecvMessage:", timeover)

	conn.SetReadDeadline(time.Now().Add(timeover))

	fourCC := make([]byte, 4)
	n, err := io.ReadFull(conn, fourCC)
	if err != nil {
		log.Println(n, err)
		return
	}
	prefix = string(fourCC)

	lengthBytes := make([]byte, 4)
	n, err = io.ReadFull(conn, lengthBytes)
	if err != nil {
		log.Println(n, err)
		return
	}

	dataLength := binary.BigEndian.Uint32(lengthBytes) // default is big endian

	data = make([]byte, dataLength)
	n, err = io.ReadFull(conn, data)
	if err != nil {
		log.Println(n, err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
func TCPParseRequest(conn net.Conn) (err error) {
	log.Println("IN TCPParseRequest:", conn.RemoteAddr())

	prefix, msg, err := TCPRecvMessage(conn, 3*time.Second)
	if err != nil {
		log.Println(err)
		return
	}

	if prefix != RSSP_MARK_RTXT {
		err = fmt.Errorf("invalid prefix for tcp request: %s", prefix)
		log.Println(err)
		return
	}

	req := string(msg)
	log.Println("URL:", req)

	uris := strings.Split(req, "?")
	if len(uris) < 2 {
		log.Println("invalid tcp req:", req)
		return
	}

	qo, err := GetQueryOptionFromString("tcp", uris[0], uris[1])
	if err != nil {
		log.Println(err)
		return
	}

	switch uris[0] {
	case "/pang/tcp/eco":
		err = PangTCPReflector(conn, qo)
	case "/pang/tcp/pub":
		err = PangTCPPublisher(conn, qo)
	case "/pang/tcp/sub":
		err = PangTCPSubscriber(conn, qo)
	case "/pang/tcp/meb":
		err = PangTCPMedusa(conn, qo)
	case "/pang/tcp/p2p":
		err = PangTCPPeering(conn, qo)
	case "/pang/tcp/tst":
		err = PangTCPTester(conn, qo)
	default:
		err = fmt.Errorf("invalid tcp path: %s", uris[0])
		log.Println(err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
func RunPlainTCPServer(pst *Studio, port int) {
	if port == 0 {
		log.Println("invalid plain tcp port:", port)
		return
	}

	w := pStudio.addNewWorkerWithParams("/server/tcp/api", pst.ID, "system")
	defer pStudio.deleteWorker(w)

	w.Addr = fmt.Sprintf(":%d", port)
	w.Proto = "tcp"
	log.Println("plain (tcp) server started on", w.Addr)

	listener, err := net.Listen("tcp", w.Addr)
	if err != nil {
		log.Println(err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			return
		}

		w.AtUsed = time.Now()

		tcpConn := conn.(*net.TCPConn)
		tcpConn.SetNoDelay(true)

		go TCPParseRequest(conn)
	}
}

// ---------------------------------------------------------------------------------
func RunSecureTCPServer(pst *Studio, port int) {
	if port == 0 {
		log.Println("invalid secure tcp port:", port)
		return
	}

	w := pStudio.addNewWorkerWithParams("/server/tcps/api", pst.ID, "system")
	defer pStudio.deleteWorker(w)

	w.Addr = fmt.Sprintf(":%d", port)
	w.Proto = "tcps"
	log.Println("secure (tcp) server started on", w.Addr)

	cert, err := tls.LoadX509KeyPair(mConfig.PEMPublic, mConfig.PEMPrivate)
	if err != nil {
		log.Fatalf("failed to load key pair: %s", err)
	}

	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	listener, err := tls.Listen("tcp", w.Addr, config)
	if err != nil {
		log.Fatalf("failed to listen: %s", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			return
		}

		w.AtUsed = time.Now()

		tcpConn := conn.(*tls.Conn)
		rawConn := tcpConn.NetConn()
		tcpRawConn := rawConn.(*net.TCPConn)
		tcpRawConn.SetNoDelay(true)

		go TCPParseRequest(conn)
	}
}

// ---------------------------------------------------------------------------------
func GetFreeTCPPort() (port int, err error) {
	taddr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		log.Println(err)
		return
	}

	l, err := net.ListenTCP("tcp", taddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer l.Close()
	port = l.Addr().(*net.TCPAddr).Port
	return
}

// ---------------------------------------------------------------------------------
func GetFreeTCPPorts(count int) (ports []int, err error) {
	for i := 0; i < count; i++ {
		taddr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			log.Println(err)
			break
		}

		l, err := net.ListenTCP("tcp", taddr)
		if err != nil {
			log.Println(err)
			break
		}
		defer l.Close()
		ports = append(ports, l.Addr().(*net.TCPAddr).Port)
	}
	return
}

//=================================================================================
