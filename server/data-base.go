// =================================================================================
// Filename: data-base.go
// Function: data structures for server operation
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-2022, 2025
// Caution: Don't use pointer method in Stringers (2020/04/01) -> Use it. Why?
// =================================================================================
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
const (
	WAIT_BASE_SECONDS      = 30                              // default 30 seconds
	WAIT_MAX_SECONDS       = 3600                            // 1 hour in seconds
	TIME_WAIT_BASE_SECONDS = WAIT_BASE_SECONDS * time.Second // in time.Duration
	TIME_WAIT_MAX_SECONDS  = WAIT_MAX_SECONDS * time.Second  // in time.Duration
)

// ---------------------------------------------------------------------------------
const (
	Idle  = 0
	Using = 1
)

type State int

func (s State) String() string {
	if s == Idle {
		return "idle"
	} else {
		return "using"
	}
}

func StringToState(str string) State {
	if str == "idle" {
		return Idle
	} else {
		return Using
	}
}

func (s State) isState(state State) bool {
	return s == state
}

// ---------------------------------------------------------------------------------
type Common struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Desc      string    `json:"desc"` // Description
	Style     string    `json:"style"`
	State     State     `json:"state"`
	Blocked   bool      `json:"blocked"`
	AtCreated time.Time `json:"at_created,omitempty"`
	AtUpdated time.Time `json:"at_updated,omitempty"`
	AtExpired time.Time `json:"at_expired,omitempty"`
	AtUsed    time.Time `json:"at_used,omitempty"`
}

func (d *Common) String() (str string) {
	str += fmt.Sprintf("[%s] ID: %20s, Name: %20s, Block: %5v State: %5s | cTime: %s, uTime: %s |",
		d.Type, d.ID, d.Name, d.Blocked, d.State.String(),
		d.AtCreated.Format("2006/01/02 15:04:05"), d.AtUsed.Format("2006/01/02 15:04:05"))
	return
}

func (d *Common) isState(state State) bool {
	return d.State == state
}

func (d *Common) setState(state State) {
	d.State = state
}

// ---------------------------------------------------------------------------------
type Handle interface {
	*websocket.Conn
}

// ---------------------------------------------------------------------------------
type Metric struct {
	BPS      float64 `json:"bps,omitempty"`
	FPS      float64 `json:"fps,omitempty"`
	InBytes  int     `json:"in_bytes,omitempty"`
	OutBytes int     `json:"out_bytes,omitempty"`
	Interval int     `json:"interval,omitempty"` // in Seconds
}

func (d *Metric) String() (str string) {
	return fmt.Sprintf("since: %v, total: (%d,%d), bps:%.2f, fps:%.2f",
		d.Interval, d.InBytes, d.OutBytes, d.BPS, d.FPS)
}

// ---------------------------------------------------------------------------------
// Session: core data structure
// ---------------------------------------------------------------------------------
type Session struct {
	Common     `json:"common,inline"`
	Metric     `json:"metric,inline"`
	RemoteAddr string        `json:"remote_addr,omitempty"`
	BridgeID   string        `json:"bridge_id,omitempty"`
	GroupID    string        `json:"group_id,omitempty"`
	ChannelID  string        `json:"channel_id,omitempty"`
	SourceID   string        `json:"source_id,omitempty"`
	TrackID    string        `json:"track_id,omitempty"`
	RequestID  string        `json:"request_id,omitempty"`
	TimeOver   time.Duration `json:"time_over"` // timeout in second
	TimeUnit   time.Duration `json:"time_unit"` // time clock in scale
	// --- internal variables
	sync.Mutex
	eventChan chan EventMessage
	req       *http.Request
	chn       *Channel
	src       *Source
	trk       *Track
	// --- connection handles for peer
	ws *websocket.Conn
}

func (d *Session) String() (str string) {
	str = d.Common.String()
	if d.req != nil {
		str += fmt.Sprintf("\n\t[client] addr: %s, uri: %s", d.req.RemoteAddr, d.req.RequestURI)
	}
	str += fmt.Sprintf("\n\tchannel: %20s, source: %10s, track: %10s, time: %s,%s",
		d.ChannelID, d.SourceID, d.TrackID, d.TimeUnit, d.TimeOver)
	str += fmt.Sprintf("\n\tbridge: %20s, group: %20s, reqid: %s", d.BridgeID, d.GroupID, d.RequestID)
	str += fmt.Sprintf("\n\tstats: %s", d.Metric.String())
	return
}

func (d *Session) newSessionValue() {
	d.State = Idle
	d.AtCreated = time.Now()
	d.AtUsed = d.AtCreated
	d.eventChan = make(chan EventMessage, 2)
}

func NewSessionPointer() (d *Session) {
	d = &Session{}
	d.Type = "session"
	d.ID = GetXidString()
	d.newSessionValue()
	return
}

func NewSessionPointerWithNameRequest(name, reqid string) (d *Session) {
	d = NewSessionPointer()
	d.State = Using
	d.Name = name
	d.RequestID = reqid
	return
}

func NewSessionPointerWithName(name string) (d *Session) {
	d = NewSessionPointer()
	d.State = Using
	d.Name = name
	return
}

func (d *Session) close() {
	d.State = Idle
	SafeCloseMessage(d.eventChan)
}

func (d *Session) setWebSocketTimeout(ws *websocket.Conn) {
	ws.SetReadDeadline(time.Now().Add(d.TimeOver))
}

func (d *Session) setTimeoutInUnit(tout int, tunit string) {
	switch tunit {
	case "zero":
		d.TimeUnit = 0
	case "nano": // ns
		d.TimeUnit = time.Nanosecond
	case "micro": // us
		d.TimeUnit = time.Microsecond
	case "milli": // ms
		d.TimeUnit = time.Millisecond
	case "sec": // s
		d.TimeUnit = time.Second
	default:
		d.TimeUnit = time.Millisecond
	}
	if tout > 0 && tout < WAIT_MAX_SECONDS {
		d.TimeOver = time.Duration(tout) * time.Second
	} else {
		d.TimeOver = TIME_WAIT_BASE_SECONDS
	}
	log.Println("settime:", tunit, tout, "=>", d.TimeUnit, d.TimeOver)
}

func (d *Session) resetTrackInfo() {
	if d.trk != nil {
		d.trk.Mime = ""
	}
	d.trk.InBytes = 0
	d.trk.OutBytes = 0
}

// ---------------------------------------------------------------------------------
// Channel is for 1:N communication
// ---------------------------------------------------------------------------------
type Channel struct {
	Common      `json:"common,inline"`
	Metric      `json:"metric,inline"`
	StreamKey   string                     `json:"-"` // `json:"stream_key"`
	Sources     map[string]*Source         `json:"-"`
	Publishers  map[string]*Session        `json:"-"`
	Subscribers map[string]*Session        `json:"-"`
	Peers       map[string]*Session        `json:"-"` // for P2P Streaming
	Eventers    map[string]*websocket.Conn `json:"-"`
	EventState  State                      `json:"-"`

	RecordAuto  bool  `json:"record_auto"`  // recording
	RecordState State `json:"record_state"` //
	TransAuto   bool  `json:"trans_auto"`   // transcoding
	TransState  State `json:"trans_state"`  //
	ProcsAuto   bool  `json:"procs_auto"`   // processing
	ProcsState  State `json:"procs_state"`  //
	RelayAuto   bool  `json:"relay_auto"`   // relaying
	RelayState  State `json:"relay_state"`  //
	ShootAuto   bool  `json:"shoot_auto"`   // shooting
	ShootState  State `json:"shoot_state"`
	// --- internal variables
	sync.Mutex
	eventChan chan EventMessage `json:"-"`
	recordCmd *exec.Cmd         `json:"-"`
	transCmd  *exec.Cmd         `json:"-"`
	procsCmd  *exec.Cmd         `json:"-"`
	relayCmd  *exec.Cmd         `json:"-"`
	shootCmd  *exec.Cmd         `json:"-"`
}

// custom json marshal
func (d *Channel) MarshalJSON() ([]byte, error) {
	d.Lock()
	defer d.Unlock()

	type Alias Channel
	return json.Marshal(&struct {
		*Alias
		Publishers  int `json:"n_pubs"`
		Subscribers int `json:"n_subs"`
	}{
		Alias:       (*Alias)(d),
		Publishers:  len(d.Publishers),
		Subscribers: len(d.Subscribers),
	})
}

func (d *Channel) String() (str string) {
	d.Lock()
	defer d.Unlock()

	str = d.Common.String()
	str += fmt.Sprintf("\n\tStyle: %8s, StreamKey: %10s, Update: %s, Expire: %s |",
		d.Style, d.StreamKey, d.AtCreated.Format("2006/01/02 15:04:05"), d.AtExpired.Format("2006/01/02 15:04:05"))
	str += fmt.Sprintf("\n\tRecord: %v, %s, Trans: %v, %s, Procs: %v, %s, Relay: %v, %s |",
		d.RecordAuto, d.RecordState.String(), d.TransAuto, d.TransState.String(),
		d.ProcsAuto, d.ProcsState.String(), d.RelayAuto, d.RelayState.String())
	str += fmt.Sprintf("\n\tstats: %s", d.Metric.String())
	str += fmt.Sprintf("\n\tPubs(%d): ", len(d.Publishers))
	for k := range d.Publishers {
		str += fmt.Sprintf("%s, ", k)
	}
	str += fmt.Sprintf("\n\tSubs(%d): ", len(d.Subscribers))
	for k := range d.Subscribers {
		str += fmt.Sprintf("%s, ", k)
	}
	return
}

func (d *Channel) newChannelValue() {
	d.Style = "static"
	d.State = Idle
	d.AtCreated = time.Now()
	d.AtUsed = d.AtCreated
	d.initChannelData()
}

func (d *Channel) setChannelValue() {
	d.State = Idle
	d.AtUsed = time.Now()
	d.EventState = Idle
	d.RecordState = Idle
	d.TransState = Idle
	d.ProcsState = Idle
	d.RelayState = Idle
	d.initChannelData()
}

func (d *Channel) initChannelData() {
	d.Publishers = make(map[string]*Session)
	d.Subscribers = make(map[string]*Session)
	d.Peers = make(map[string]*Session) // for experiments
	d.Eventers = make(map[string]*websocket.Conn)
	d.Sources = make(map[string]*Source)
	d.eventChan = make(chan EventMessage, 2)
}

func NewChannelPointer() (d *Channel) {
	d = &Channel{}
	d.Type = "channel"
	d.ID = GetXidString()
	d.newChannelValue()
	return
}

func (d *Channel) close() {
	d.purgeAllSourceTracks()
	SafeCloseMessage(d.eventChan)
}

func (d *Channel) reset() {
	d.resetAllSourceTracks()
}

func (d *Channel) isValidStreamKey(key string) bool {
	return d.StreamKey == "" || d.StreamKey == key
}

// ---------------------------------------------------------------------------------
func (d *Channel) addPublisher(s *Session) {
	d.Lock()
	defer d.Unlock()

	d.Publishers[s.ID] = s
	d.AtUpdated = time.Now()
}

func (d *Channel) deletePublisher(s *Session) {
	d.Lock()
	defer d.Unlock()

	delete(d.Publishers, s.ID)
}

// ---------------------------------------------------------------------------------
func (d *Channel) addSubscriber(s *Session) {
	d.Lock()
	defer d.Unlock()

	d.Subscribers[s.ID] = s
	d.AtUsed = time.Now()
}

func (d *Channel) deleteSubscriber(s *Session) {
	d.Lock()
	defer d.Unlock()

	delete(d.Subscribers, s.ID)
}

// ---------------------------------------------------------------------------------
// add/deleteSource don't use the lock to avoid deadlock due to double locks
func (d *Channel) findSourceByLabel(slabel string) (s *Source) {
	return d.Sources[slabel]
}

func (d *Channel) addSourceByLabel(slabel string) (s *Source) {
	d.Sources[slabel] = NewSource(slabel)
	s = d.Sources[slabel]
	return
}

func (d *Channel) deleteSourceByLabel(slabel string) (s *Source) {
	delete(d.Sources, slabel)
	return
}

// ---------------------------------------------------------------------------------
func (d *Channel) findSourceTrackByLabel(slabel, tlabel string) (s *Source, t *Track, err error) {
	d.Lock()
	defer d.Unlock()

	s = d.Sources[slabel]
	if s == nil {
		err = fmt.Errorf("no source %s error", slabel)
		return
	}
	t = s.Tracks[tlabel]
	if t == nil {
		err = fmt.Errorf("no track %s error", tlabel)
		return
	}
	return
}

func (d *Channel) addSourceTrackByLabel(slabel, tlabel string) (s *Source, t *Track, err error) {
	d.Lock()
	defer d.Unlock()

	s = d.Sources[slabel] // already the source is allocated?
	if s == nil {
		s = d.addSourceByLabel(slabel) // add the source if not exist
		if s == nil {
			err = fmt.Errorf("add source %s error", slabel)
			return
		}
	}
	t = s.Tracks[tlabel] // already the track is allocated?
	if t == nil {
		t = s.addTrackByLabel(tlabel, BUFFER_CAP_SLOTS, BUFFER_LEN_SLOTS) // add the track if not exist
		if t == nil {
			err = fmt.Errorf("add track %s error", tlabel)
			return
		}
	}
	return
}

// func (d *Channel) deleteSourceTrackByLabel(slabel, tlabel string) (err error) {
// 	d.Lock()
// 	defer d.Unlock()

// 	if !d.isIdle() {
// 		err = fmt.Errorf("channel state %s error", d.State)
// 		return
// 	}
// 	s := d.Sources[slabel]
// 	if s == nil {
// 		err = fmt.Errorf("no source %s error", slabel)
// 		return
// 	}
// 	// remove tracks of the source at first, and remove the source
// 	for _, t := range s.Tracks {
// 		s.deleteTrackByLabel(t.Label)
// 	}
// 	d.deleteSourceByLabel(s.Label)
// 	return
// }

func (d *Channel) purgeAllSourceTracks() (err error) {
	d.Lock()
	defer d.Unlock()

	if !d.isState(Idle) {
		err = fmt.Errorf("channel state %s error", d.State)
		return
	}
	// remove sources and their tracks of channel
	for _, s := range d.Sources {
		for _, t := range s.Tracks {
			s.deleteTrackByLabel(t.Label)
		}
		d.deleteSourceByLabel(s.Label)
	}
	return
}

func (d *Channel) resetAllSourceTracks() (err error) {
	d.Lock()
	defer d.Unlock()

	if !d.isState(Idle) {
		err = fmt.Errorf("channel state %s error", d.State)
		return
	}
	// reset sources and their tracks to have their initial info
	for _, s := range d.Sources {
		for _, t := range s.Tracks {
			t.Style = "mono"
			t.Mime = ""
		}
	}
	return
}

func (d *Channel) ListResources(cs, ct string) (str string) {
	d.Lock()
	defer d.Unlock()

	str += fmt.Sprintf("[channel] %s, %s, %s (%d,%d)\n", d.ID, d.Name, d.State, d.InBytes, d.OutBytes)
	for _, s := range d.Sources {
		if cs != "" {
			if !strings.Contains(s.Label, cs) {
				continue
			}
		}
		str += fmt.Sprintf("\t[source] %s, %s:\n", s.ID, s.Label)
		for _, t := range s.Tracks {
			if ct != "" {
				if !strings.Contains(t.Label, ct) {
					continue
				}
			}
			str += fmt.Sprintf("\t\t[track] %s, %s: %s,%s, %d (%d,%d) %s\n",
				t.ID, t.Label, t.Mode, t.Style, t.Num, t.InBytes, t.OutBytes, t.ProcName)
			str += fmt.Sprintf("\t\t\tMIME: %s\n", t.Mime)
			for i, b := range t.Rings {
				str += fmt.Sprintf("\t\t\t[%d] %s, N:%d/%2d W:%d,R:%d\tmime:%s\n",
					i, b.Label, b.SizeLen, b.SizeCap, b.PosRead, b.PosWrite, b.Mime)
			}
			str += "\n"
		}
	}
	return
}

// ---------------------------------------------------------------------------------
func (d *Channel) pushEvent(name, data, path, reqid string) {
	log.Println("pushEvent", name, data, path, reqid)

	em := EventMessage{Type: "event", ID: GetXidString(),
		Name: name, Data: data, Path: path, RequestID: reqid, AtCreated: time.Now()}
	if d.isEventState(Using) { // don't send when channel event handler is not ready
		d.eventChan <- em
	}
	if pStudio.isEventState(Using) { // don't send when studio event handler is not ready
		if name == "pub-in" { // toss this to the system event
			pStudio.eventChan <- em
			log.Println(em)
		}
	}
}

func (d *Channel) addEventer(id string, ws *websocket.Conn) {
	d.Lock()
	defer d.Unlock()
	d.Eventers[id] = ws
}

func (d *Channel) deleteEventer(id string) {
	d.Lock()
	defer d.Unlock()
	delete(d.Eventers, id)
}

func (d *Channel) isNoEventers() bool {
	d.Lock()
	defer d.Unlock()
	return len(d.Eventers) == 0
}

func (d *Channel) isEventState(state State) bool {
	d.Lock()
	defer d.Unlock()
	return d.EventState == state
}

func (d *Channel) setEventState(state State) {
	d.Lock()
	defer d.Unlock()
	d.EventState = state
}

// ---------------------------------------------------------------------------------
// Worker for system worker thread
// ---------------------------------------------------------------------------------
type Worker struct {
	Common    `json:"common,inline"`
	SessionID string `json:"session_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
	GroupID   string `json:"group_id,omitempty"`
	Proto     string `json:"proto,omitempty"`
	Addr      string `json:"addr,omitempty"`
	Style     string `json:"style,omitempty"`
	// --- internal variables
	sync.Mutex
}

func (d *Worker) String() (str string) {
	str = d.Common.String()
	str += fmt.Sprintf("\n\tsession: %20s, channel: %20s, group: %20s, addr: %s, proto: %s, style: %s\n",
		d.SessionID, d.ChannelID, d.GroupID, d.Addr, d.Proto, d.Style)
	return
}

func (d *Worker) newWorkerValue() {
	d.State = Idle
	d.AtCreated = time.Now()
	d.AtUsed = d.AtCreated
	d.initWorkerValue()
}

// func (d *Worker) setWorkerValue() {
// 	d.State = Idle
// 	d.AtUsed = time.Now()
// 	d.initWorkerValue()
// }

func (d *Worker) initWorkerValue() {
	d.ChannelID = ""
}

func NewWorkerPointer() (d *Worker) {
	d = &Worker{}
	d.Type = "worker"
	d.ID = GetXidString()
	d.newWorkerValue()
	return
}

func NewWorkerPointerWithParams(name, chid, style string) (d *Worker) {
	d = NewWorkerPointer()
	d.Name = name
	d.ChannelID = chid
	d.State = Using
	d.Style = style
	return
}

// ---------------------------------------------------------------------------------
// Punch for information using hole punching between peers
// ---------------------------------------------------------------------------------
type Punch struct {
	Common     `json:"common,inline"`
	SessionID  string `json:"session_id,omitempty"`
	ChannelID  string `json:"channel_id,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
	TrackID    string `json:"track_id,omitempty"`
	ResourceID string `json:"resource_id,omitempty"` // channel/source/track id
	Role       string `json:"role,omitempty"`        // role: pub, sub, meb
	Addr       string `json:"addr,omitempty"`        // address for hole punching
	// --- internal variables
	sync.Mutex
}

func (d *Punch) String() (str string) {
	str = d.Common.String()
	str += fmt.Sprintf("\n\tsession: %20s, channel: %20s, source: %s, track: %s",
		d.SessionID, d.ChannelID, d.SourceID, d.TrackID)
	str += fmt.Sprintf("\n\tresource: %s, role: %s, addr: %s\n", d.ResourceID, d.Role, d.Addr)
	return
}

func (d *Punch) newPunchValue() {
	d.State = Idle
	d.AtCreated = time.Now()
	d.AtUsed = d.AtCreated
	d.initPunchValue()
}

// func (d *Worker) setPunchValue() {
// 	d.State = Idle
// 	d.AtUsed = time.Now()
// 	d.initPunchValue()
// }

func (d *Punch) initPunchValue() {
	d.ChannelID = ""
}

func NewPunchPointer() (d *Punch) {
	d = &Punch{}
	d.Type = "punch"
	d.ID = GetXidString()
	d.newPunchValue()
	return
}

func NewPunchPointerWithParams(chid, srid, tkid string) (d *Punch) {
	d = NewPunchPointer()
	d.Name = fmt.Sprintf("/%s/%s/%s", chid, srid, tkid)
	d.ChannelID = chid
	d.SourceID = srid
	d.TrackID = tkid
	d.State = Using
	return
}

//=================================================================================
