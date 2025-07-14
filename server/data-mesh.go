// =================================================================================
// Filename: data-mesh.go
// Function: data structure and its functions for mesh handling
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2023 - 2025
// =================================================================================
package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
)

// ---------------------------------------------------------------------------------
// Group for set of channels and sessions (Swarm)
// ---------------------------------------------------------------------------------
type Group struct {
	Common   `json:"common,inline"`
	Channels map[string]*Channel `json:"channels,omitempty"`
	Sessions map[string]*Session `json:"sessions,omitempty"`
	// --- internal variables
	sync.Mutex
}

func (d *Group) String() (str string) {
	str = d.Common.String()
	str += "\n\t[channel] "
	for k := range d.Channels {
		str += fmt.Sprintf("%s ", k)
	}
	str += "\n\t[session] "
	for k := range d.Sessions {
		str += fmt.Sprintf("%s ", k)
	}
	return
}

func (d *Group) newGroupValue() {
	d.State = Idle
	d.AtCreated = time.Now()
	d.AtUsed = d.AtCreated
	d.initGroupValue()
}

func (d *Group) setGroupValue() {
	d.State = Idle
	d.AtUsed = time.Now()
	d.initGroupValue()
}

func (d *Group) initGroupValue() {
	d.Sessions = make(map[string]*Session)
	d.Channels = make(map[string]*Channel)
}

func NewGroupPointer() (d *Group) {
	d = &Group{}
	d.Type = "group"
	d.ID = GetXidString()
	d.newGroupValue()
	return
}

func NewGroupPointerWithName(name string) (d *Group) {
	d = NewGroupPointer()
	d.Name = name
	d.State = Using
	return
}

//---------------------------------------------------------------------------------
// Studio: Group
//---------------------------------------------------------------------------------

// ..............................................................
// Concurrency control : Group
// ..............................................................
func (d *Studio) findGroupByID(pid string) (r *Group) {
	d.GroupGate.RLock()
	defer d.GroupGate.RUnlock()

	r = d.Groups[pid]
	return
}

func (d *Studio) addGroup(p *Group) (r *Group) {
	d.GroupGate.Lock()
	defer d.GroupGate.Unlock()

	d.Groups[p.ID] = p
	r = d.Groups[p.ID]
	return
}

func (d *Studio) deleteGroup(p *Group) {
	d.GroupGate.Lock()
	defer d.GroupGate.Unlock()

	delete(d.Groups, p.ID)
}

// ---------------------------------------------------------------------------------
// start and enc points of bridge
type Spot struct {
	Proto       string `json:"proto,omitempty"`
	Addr        string `json:"addr,omitempty"`
	API         string `json:"api,omitempty"`
	ChannelID   string `json:"chid,omitempty"`
	SourceLabel string `json:"slabel,omitempty"`
	TrackLabel  string `json:"tlabel,omitempty"`
}

func (d *Spot) String() (str string) {
	str = fmt.Sprintf("%s://%s%s (%s / %s / %s)",
		d.Proto, d.Addr, d.API, d.ChannelID, d.SourceLabel, d.TrackLabel)
	return
}

// Connection line between tracks of two channels
type Bridge struct {
	Common    `json:"common,inline"`
	Metric    `json:"metric,inline"`
	RequestID string `json:"request_id,omitempty"`
	Timeout   int    `json:"timeout,omitempty"` // timeout in second
	// --- bridge specific
	From       Spot   `json:"from,omitempty"`
	To         Spot   `json:"to,omitempty"`
	Attr       string `json:"attr"`                // auto, ever, none(manual)
	Direction  string `json:"direction,omitempty"` // push(default) or pull
	TransCodec string `json:"trans_codec,omitempty"`
}

func (d *Bridge) String() (str string) {
	str = d.Common.String()
	str += fmt.Sprintf("\n\ttimeout: %ds, attr: %s, trans: %s, direction: %s", d.Timeout, d.Attr, d.TransCodec, d.Direction)
	str += fmt.Sprintf("\n\tfrom: %s", d.From.String())
	str += fmt.Sprintf("\n\t=>to: %s", d.To.String())
	return
}

func (d *Bridge) newBridgeValue() {
	d.State = Idle
	d.AtCreated = time.Now()
	d.AtUsed = d.AtCreated
	d.Attr = "none"
	d.initBridgeValue()
}

func (d *Bridge) setBridgeValue() {
	d.State = Idle
	d.AtUsed = time.Now()
	if d.From.Proto == "" {
		d.From.Proto = "int" // default protocol of From
	}
	if d.To.Proto == "" {
		d.To.Proto = "ws" // default protocol of To
	}
	d.initBridgeValue()
}

func (d *Bridge) initBridgeValue() {
	if d.Attr == "" {
		d.Attr = "none"
	}
	if d.Direction == "" {
		d.Direction = "push"
	}
	if d.Timeout < 1 {
		d.Timeout = 10 // default timeout in second
	}
}

func NewBridgePointer() (d *Bridge) {
	d = &Bridge{}
	d.Type = "bridge"
	d.ID = GetXidString()
	d.newBridgeValue()
	return
}

func NewBridgePointerWithNameRequest(name, reqid string) (d *Bridge) {
	d = NewBridgePointer()
	d.State = Using
	d.Name = name
	d.RequestID = reqid
	return
}

func NewBridgePointerWithName(name string) (d *Bridge) {
	d = NewBridgePointer()
	d.State = Using
	d.Name = name
	return
}

// ---------------------------------------------------------------------------------
// Studio: Bridge
// ---------------------------------------------------------------------------------
func (d *Studio) checkBridges(attr string) (err error) {
	d.BridgeGate.RLock()
	defer d.BridgeGate.RUnlock()

	for _, v := range d.Bridges {
		if v.Blocked {
			continue
		}
		if v.Attr == attr {
			err = d.startBridge(v)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
	return
}

// ---------------------------------------------------------------------------------
func (d *Studio) startBridge(b *Bridge) (err error) {
	log.Println("i.startBridge", b.ID, b.Name)

	if b.Direction == "pull" {
		err = d.startPullBridge(b)
	} else {
		err = d.startPushBridge(b) // push is default type
	}
	return
}

func (d *Studio) startPushBridge(b *Bridge) (err error) {
	log.Println("i.startPushBridge", b.ID, b.Name)

	chn := d.findChannelByIDState(b.From.ChannelID, Using)
	if chn == nil {
		err = fmt.Errorf("not found proper push channel: %s", b.From.ChannelID)
		return
	}

	for _, src := range chn.Sources {
		if b.From.SourceLabel != "" && b.From.SourceLabel != src.Label {
			// filter sources if its label not matched
			continue
		}
		for _, trk := range src.Tracks {
			if trk.Mime == "" {
				continue
			}
			if b.From.TrackLabel != "" && b.From.TrackLabel != trk.Label {
				// filter tracks if its label not matched
				continue
			}
			frpath := fmt.Sprintf("%s://%s%s", b.From.Proto, b.From.Addr, b.From.API)
			frqry := fmt.Sprintf("channel=%s&source=%s&track=%s", b.From.ChannelID, src.Label, trk.Label)
			frqry += fmt.Sprintf("&timeout=%d", b.Timeout) // default is 10 seconds
			topath := fmt.Sprintf("%s://%s%s", b.To.Proto, b.To.Addr, b.To.API)
			toqry := fmt.Sprintf("channel=%s", b.To.ChannelID)
			// change source and track label if needed
			if b.To.SourceLabel == "" {
				toqry += fmt.Sprintf("&source=%s", src.Label)
			} else {
				toqry += fmt.Sprintf("&source=%s", b.To.SourceLabel)
			}
			if b.To.TrackLabel == "" {
				toqry += fmt.Sprintf("&track=%s", trk.Label)
			} else {
				toqry += fmt.Sprintf("&track=%s", b.To.TrackLabel)
			}
			toqry += fmt.Sprintf("&timeout=%d", b.Timeout) // default is 10 seconds
			toqry += fmt.Sprintf("&mode=%s", trk.Mode)
			go b.pushSourceTrack(frpath, frqry, topath, toqry)
		}
	}
	return
}

func (d *Studio) startPullBridge(b *Bridge) (err error) {
	log.Println("i.startPullBridge", b.ID, b.Name)

	sources, err := b.getResourceListByChannelID(b.From.ChannelID)
	if err != nil {
		log.Println(err)
		return
	}

	chn := d.findChannelByID(b.To.ChannelID)
	if chn == nil {
		err = fmt.Errorf("not found proper pull channel: %s", b.To.ChannelID)
		return
	}

	for _, src := range sources {
		if b.From.SourceLabel != "" && b.From.SourceLabel != src.Label {
			continue
		}
		for _, trk := range src.Tracks {
			if trk.Mime == "" {
				continue
			}
			if b.From.TrackLabel != "" && b.From.TrackLabel != trk.Label {
				continue
			}
			frpath := fmt.Sprintf("%s://%s%s", b.From.Proto, b.From.Addr, b.From.API)
			frqry := fmt.Sprintf("channel=%s&source=%s&track=%s", b.From.ChannelID, src.Label, trk.Label)
			frqry += fmt.Sprintf("&timeout=%d", b.Timeout)
			topath := fmt.Sprintf("%s://%s%s", b.To.Proto, b.To.Addr, b.To.API)
			toqry := fmt.Sprintf("channel=%s", b.To.ChannelID)
			// change source and track label if needed
			if b.To.SourceLabel == "" {
				toqry += fmt.Sprintf("&source=%s", src.Label)
			} else {
				toqry += fmt.Sprintf("&source=%s", b.To.SourceLabel)
			}
			if b.To.TrackLabel == "" {
				toqry += fmt.Sprintf("&track=%s", trk.Label)
			} else {
				toqry += fmt.Sprintf("&track=%s", b.To.TrackLabel)
			}
			toqry += fmt.Sprintf("&timeout=%d", b.Timeout)
			toqry += fmt.Sprintf("&mode=%s", trk.Mode)
			go b.pullSourceTrack(frpath, frqry, topath, toqry)
		}
	}
	return
}

func (d *Studio) stopBridge(b *Bridge) (err error) {
	log.Println("i.stopBridge", b.ID, b.Name)

	ses := d.findSessionByBridgeID(b.ID)
	if ses == nil {
		err = fmt.Errorf("not found session of bridge: %s", b.ID)
		return
	}
	ses.close()
	return
}

// ---------------------------------------------------------------------------------
func (b *Bridge) pushSourceTrack(frpath, frqry, topath, toqry string) (err error) {
	log.Println("i.pushSourceTrack", frpath, frqry, topath, toqry)

	qo, err := GetQueryOptionFromString("int", frpath, frqry)
	if err != nil {
		log.Println(err)
		return
	}
	qo.URL.Path = b.From.API
	log.Println(qo)

	url := fmt.Sprintf("%s?%s", topath, toqry)
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	err = PangWSPusher(b, ws, qo)
	return
}

func (b *Bridge) pullSourceTrack(frpath, frqry, topath, toqry string) (err error) {
	log.Println("i.pullSourceTrack", frpath, frqry, topath, toqry)

	url := fmt.Sprintf("%s?%s", frpath, frqry)
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Println(err)
		return
	}
	defer ws.Close()

	qo, err := GetQueryOptionFromString("ws", topath, toqry)
	if err != nil {
		log.Println(err)
		return
	}
	qo.URL.Path = b.To.API
	log.Println(qo)

	err = PangWSPuller(b, ws, qo)
	return
}

func (b *Bridge) getResourceListByChannelID(chid string) (sources map[string]*Source, err error) {
	log.Println("i.getResourceListByChannelID", chid)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	url := fmt.Sprintf("http://%s/manager/http/cmd?op=show&obj=channel&id=%s&opt=track&format=json", b.From.Addr, chid)
	res, err := client.Get(url)
	if err != nil {
		log.Println(res, err)
		return
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println(string(body), err)
		return
	}

	err = json.Unmarshal(body, &sources)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// ..............................................................
// Concurrency control : Bridge
// ..............................................................
func (d *Studio) setBridgeByIDState(id string, state State) (r *Bridge) {
	d.BridgeGate.RLock()
	defer d.BridgeGate.RUnlock()

	r = d.Bridges[id]
	if r == nil {
		return nil
	}
	r.State = state
	return
}

func (d *Studio) findBridgeByID(pid string) (r *Bridge) {
	d.BridgeGate.RLock()
	defer d.BridgeGate.RUnlock()

	r = d.Bridges[pid]
	return
}

func (d *Studio) addBridge(p *Bridge) (r *Bridge) {
	d.BridgeGate.Lock()
	defer d.BridgeGate.Unlock()

	d.Bridges[p.ID] = p
	r = d.Bridges[p.ID]
	return
}

func (d *Studio) deleteBridge(p *Bridge) {
	d.BridgeGate.Lock()
	defer d.BridgeGate.Unlock()

	delete(d.Bridges, p.ID)
}

//=================================================================================
