// =================================================================================
// Filename: data-buffer.go
// Function: Handling Slot, Buffer, Track and Slot in a circular buffer
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2020-2022
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
const (
	TRACK_MIN_BUFFERS = 2  // min number of buffers (pipes), 2(default)
	TRACK_MAX_BUFFERS = 10 // max number of buffers (pipes)
	BUFFER_MIN_SLOTS  = 2  // min number of slots, 2(default)
	BUFFER_MAX_SLOTS  = 30 // max number of slots, 30(default)
	BUFFER_LEN_SLOTS  = 20 // 2 - 30, 15(default)
	BUFFER_CAP_SLOTS  = 30 // 30fps, 1sec
	BUFFER_GAP_SLOTS  = 2  // minimum distansce between rpos and wpos
	BUFFER_NUM_FORE   = 0  // forward buffer index
	BUFFER_NUM_BACK   = 1  // backward buffer index
)

// ---------------------------------------------------------------------------------
type Head struct {
	FrameType int       `json:"frame_type,omitempty"` // frameType
	From      string    `json:"from,omitempty"`       // from (source) id
	To        string    `json:"to,omitempty"`         // to (destination) id
	Time      time.Time `json:"time,omitempty"`       // buffering time
	Length    int       `json:"length,omitempty"`     // size of Data
}

func (d *Head) String() (str string) {
	str += fmt.Sprintf("From: %s, To: %s, Length: %d", d.From, d.To, d.Length)
	return
}

//---------------------------------------------------------------------------------
// -- 1st structure (old)
// type UnitBuffer struct {
// 	FrameType int                  // frameType : Key(0), Delta(1) in VP, I(0), P(1), B(2) in MPEG
// 	Header    textproto.MIMEHeader // multipart mime header (internal)
// 	Mime      string               // mime type of Data
// 	Length    int                  // size of Data
// 	Data      []byte               // binary data
// }

// -- 2nd new structure
type Slot struct {
	FrameType int         `json:"frame_type,omitempty"` // frameType : binary, text
	Head      interface{} `json:"head,omitempty"`       // multipart mime header (internal)
	To        string      `json:"to,omitempty"`         // mime type of Data
	Mime      string      `json:"mime,omitempty"`       // mime type of Data
	Time      time.Time   `json:"time,omitempty"`       // buffering time
	Length    int         `json:"length,omitempty"`     // size of Data
	Mark      string      `json:"mark,omitempty"`       // mark of Data
	Data      []byte      `json:"data,omitempty"`       // binary data, itself
}

func (d *Slot) String() (str string) {
	str += fmt.Sprintf("From: %v, To: %s, Length: %d", d.Head, d.To, d.Length)
	return
}

func (d *Slot) getLength() (size int) {
	d.Length = len(d.Data)
	return d.Length
}

func (d *Slot) getLengthTime() (size int) {
	d.Length = len(d.Data)
	d.Time = time.Now()
	return d.Length
}

// ---------------------------------------------------------------------------------
// Pipe is a go channel with slots
// ---------------------------------------------------------------------------------
type Pipe struct {
	ID      string    `json:"id"`       // from: xid
	Label   string    `json:"label"`    // -> data mime?
	Mime    string    `json:"mime"`     // mime type of stream, not used now!
	SizeLen int       `json:"size_len"` // number of slots currently used
	SizeCap int       `json:"size_cap"` // number of slots allocated
	Chain   chan Slot `json:"-"`        // slots to record buffer data
	// --- internal variables
	sync.RWMutex
}

// ---------------------------------------------------------------------------------
// Buffer is a kind of circular buffer with slots
// ---------------------------------------------------------------------------------
type Buffer struct {
	ID       string `json:"id"`        // from: xid
	Label    string `json:"label"`     // -> data mime?
	Mime     string `json:"mime"`      // mime type of stream, not used now!
	PosRead  int    `json:"pos_read"`  // read position
	PosWrite int    `json:"pos_write"` // write position
	SizeLen  int    `json:"size_len"`  // number of slots currently used
	SizeCap  int    `json:"size_cap"`  // number of slots allocated
	Slots    []Slot `json:"-"`         // slots to record buffer data
	// --- internal variables
	sync.RWMutex
}

func (d *Buffer) String() (str string) {
	str += fmt.Sprintf("%s Num: %d/%d, Pos: %d,%d", d.Label, d.SizeLen, d.SizeCap, d.PosRead, d.PosWrite)
	return
}

func NewBuffer(label string, n, m int) (b *Buffer) {
	b = &Buffer{
		ID:      GetXidString(),
		Label:   label,
		SizeCap: n,
		SizeLen: m,
		Slots:   make([]Slot, n),
	}
	return
}

func (d *Buffer) setBufferSizeLen(sz int) {
	if sz > 1 && sz <= d.SizeCap {
		d.SizeLen = sz
	}
}

func (d *Buffer) readSlotByPos(pos int) (b Slot) {
	b = d.Slots[pos]
	// log.Println(d.PosRead, d.PosWrite, b.Header)
	return
}

func (d *Buffer) writeSlot(b Slot, flock bool) {
	if flock { // multi-use case of buffer such as meb
		d.Lock()
		defer d.Unlock()
	}
	d.Slots[d.PosWrite] = b
	d.PosRead = d.PosWrite
	d.PosWrite = (d.PosWrite + 1) % d.SizeLen
	// log.Println(d.PosRead, d.PosWrite, b.Header)
}

func (d *Buffer) setReadPos(lpos int) (rpos int) {
	if Abs(d.PosWrite-lpos) > BUFFER_GAP_SLOTS {
		rpos = d.PosWrite
	} else {
		rpos = (lpos + 1) % d.SizeLen // CAUTION: lpos != trk.RPos
	}
	return
}

// ---------------------------------------------------------------------------------
// Track : array of ring buffers
// ---------------------------------------------------------------------------------
type Track struct {
	ID       string                      `json:"id"`              // track id
	Label    string                      `json:"label"`           // track label assigned by user
	Mime     string                      `json:"mime"`            // mime type of stream
	Mode     string                      `json:"mode"`            // single(default), bundle
	Style    string                      `json:"style"`           // mono(default), multi
	Num      int                         `json:"num"`             // number of buffers
	Rings    []*Buffer                   `json:"rings,omitempty"` // forward[0], backward[1], ...
	Hands    []*websocket.Conn           `json:"-"`               // Not used
	Zebs     map[*websocket.Conn]*Buffer `json:"-"`               // testing ...
	Cards    map[string]string           `json:"cards,omitempty"` // agent card information
	InBytes  int                         `json:"in_bytes,omitempty"`
	OutBytes int                         `json:"out_bytes,omitempty"`
	ProcName string                      `json:"proc_name,omitempty"`
	Metric   `json:"metric"`
	// --- internal variables
	sync.RWMutex
}

func (d *Track) String() (str string) {
	d.Lock()
	defer d.Unlock()
	str += fmt.Sprintf("[%s] %s:%s,%s %d (%d,%d)", d.ID, d.Label, d.Mime, d.Mode, d.Num, d.InBytes, d.OutBytes)
	for _, r := range d.Rings {
		str += r.String() + "\t"
	}
	return
}

// make a new track for bundle use of bi-directional communication
func NewTrackDualBuffers(tlabel string, n, m int) (t *Track) {
	t = &Track{ID: GetXidString(), Label: tlabel, Num: 2}
	t.Rings = append(t.Rings, NewBuffer("fore", n, m))
	t.Rings = append(t.Rings, NewBuffer("back", n, m)) // to use minimum slots
	// t.Chain = make(chan Slot, 1)
	return
}

// make a new track for parallel use of multiple buffers
func NewTrackMultipleBuffers(tlabel string, n, m, k int) (t *Track) {
	t = &Track{ID: GetXidString(), Label: tlabel, Num: k}
	for i := 0; i < k; i++ {
		blabel := fmt.Sprintf("no%02d", i)
		t.Rings = append(t.Rings, NewBuffer(blabel, n, m))
		t.Hands = append(t.Hands, nil)
	}
	// t.Chain = make(chan Slot, 1)
	return
}

func (d *Track) setTrackBufferSize(blen int) {
	d.Lock()
	defer d.Unlock()
	if blen < BUFFER_MIN_SLOTS || blen > BUFFER_MAX_SLOTS {
		log.Println("invalid buffer size:", blen)
		return
	}
	for _, r := range d.Rings {
		r.setBufferSizeLen(blen)
	}
}

func (d *Track) resetTrackInfo() {
	d.Lock()
	defer d.Unlock()
	d.Mode = ""
	d.Label = ""
	d.Mime = ""
}

// ---------------------------------------------------------------------------------
func NewTrackZebBuffers(tlabel string, n, m int) (t *Track) {
	t = &Track{ID: GetXidString(), Label: tlabel, Num: 0}
	t.Zebs = make(map[*websocket.Conn]*Buffer)
	return
}

func (d *Track) addZebBuffer(ws *websocket.Conn, blabel string, n, m int) (b *Buffer) {
	d.Lock()
	defer d.Unlock()
	if d.Zebs == nil {
		d.Zebs = make(map[*websocket.Conn]*Buffer)
	}
	b = NewBuffer(blabel, n, m)
	d.Zebs[ws] = b
	d.Num++
	return
}

func (d *Track) deleteZebBuffer(ws *websocket.Conn, b *Buffer) {
	d.Lock()
	defer d.Unlock()
	delete(d.Zebs, ws)
	d.Num--
}

func (d *Track) printZebBuffers() {
	d.Lock()
	defer d.Unlock()
	for _, b := range d.Zebs {
		fmt.Printf("%s N:%d, Z:%d\n", b.Label, d.Num, len(d.Zebs))
	}
}

// ---------------------------------------------------------------------------------
// Source : a set of Tracks for audio, video, image, graphic, text, (any) ...
// ---------------------------------------------------------------------------------
type Source struct {
	ID     string            `json:"id"`    // source id
	Label  string            `json:"label"` // source label such as base, extra
	Num    int               `json:"num"`   // number of tracks
	Tracks map[string]*Track `json:"tracks,omitempty"`
	// --- internal variables
	sync.RWMutex
}

func (d *Source) String() (str string) {
	d.Lock()
	defer d.Unlock()
	str += fmt.Sprintf("[%s] %s : ", d.ID, d.Label)
	for _, t := range d.Tracks {
		str += fmt.Sprintf("[%s] %s ", t.ID, t.Label)
	}
	return
}

func NewSource(label string) (s *Source) {
	s = &Source{ID: GetXidString(), Label: label}
	s.Tracks = make(map[string]*Track)
	return
}

// func (d *Source) findTrackByLabel(tlabel string) (rb *Track) {
// 	d.Lock()
// 	defer d.Unlock()
// 	return d.Tracks[tlabel]
// }

func (d *Source) addTrackByLabel(tlabel string, ncap, nuse int) (t *Track) {
	d.Lock()
	defer d.Unlock()
	t = d.Tracks[tlabel]
	if t != nil {
		log.Println("already allocated track", tlabel)
		return
	}

	// t = NewTrackDualBuffers(tlabel, BUFFER_CAP_SLOTS, BUFFER_LEN_SLOTS)
	t = NewTrackDualBuffers(tlabel, ncap, nuse)
	d.Tracks[tlabel] = t
	d.Num = len(d.Tracks)
	return
}

func (d *Source) deleteTrackByLabel(tlabel string) {
	d.Lock()
	defer d.Unlock()
	delete(d.Tracks, tlabel)
	d.Num = len(d.Tracks)
}

//=================================================================================
