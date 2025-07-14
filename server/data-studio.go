// =================================================================================
// Filename: data-studio.go
// Function: studio structures of server
// Copyright: TeamGRIT, 2020-2021, 2025
// Author: Stoney Kang, sikang@teamgrit.kr
// =================================================================================
package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
)

const (
	TIME_EXPIRE_DYNAMIC = 24 * time.Hour
	TIME_EXPIRE_INSTANT = 1 * time.Hour
)

// ---------------------------------------------------------------------------------
type Studio struct {
	sync.Mutex
	Common      `json:"common"`
	NumSessions [2]int                     `json:"num_sessions"`
	SessionGate sync.RWMutex               `json:"-"`
	Sessions    map[string]*Session        `json:"-"`
	NumChannels [2]int                     `json:"num_channels"`
	ChannelGate sync.RWMutex               `json:"-"`
	Channels    map[string]*Channel        `json:"-"`
	NumGroups   [2]int                     `json:"num_groups"`
	GroupGate   sync.RWMutex               `json:"-"`
	Groups      map[string]*Group          `json:"-"`
	NumPunches  [2]int                     `json:"num_punches"` // New in v1.1.6.5
	PunchGate   sync.RWMutex               `json:"-"`
	Punches     map[string]*Punch          `json:"-"`
	NumBridges  [2]int                     `json:"num_bridges"`
	BridgeGate  sync.RWMutex               `json:"-"`
	Bridges     map[string]*Bridge         `json:"-"`
	NumWorkers  [2]int                     `json:"num_workers"`
	WorkerGate  sync.RWMutex               `json:"-"`
	Workers     map[string]*Worker         `json:"-"`
	NumEventers [2]int                     `json:"num_eventers"`
	EventerGate sync.RWMutex               `json:"-"`
	Eventers    map[string]*websocket.Conn `json:"-"`
	eventChan   chan EventMessage          `json:"-"`
	EventState  State                      `json:"event_state"` // New in v1.1.7.1
}

func (d *Studio) String() (str string) {
	d.Lock()
	defer d.Unlock()

	str += d.Common.String() + "\n"
	str += fmt.Sprintf("\tsessions: %d/%d", d.NumSessions[1], d.NumSessions[0])
	str += fmt.Sprintf("\tchannels: %d/%d", d.NumChannels[1], d.NumChannels[0])
	str += fmt.Sprintf("\tgroups: %d/%d", d.NumGroups[1], d.NumGroups[0])
	str += fmt.Sprintf("\tbridges: %d/%d", d.NumBridges[1], d.NumBridges[0])
	str += fmt.Sprintf("\tworkers: %d/%d", d.NumWorkers[1], d.NumWorkers[0])
	str += fmt.Sprintf("\tpunches: %d/%d", d.NumPunches[1], d.NumPunches[0]) // New in v1.1.6.5
	return
}

func (d *Studio) initStudioValue() {
	d.Type = "studio"
	d.State = Idle
	d.AtCreated = time.Now()
	d.AtUsed = d.AtCreated
	d.Sessions = make(map[string]*Session)
	d.Channels = make(map[string]*Channel)
	d.Groups = make(map[string]*Group)
	d.Bridges = make(map[string]*Bridge)
	d.Workers = make(map[string]*Worker)
	d.Punches = make(map[string]*Punch) // New in v1.1.6.5
	d.Eventers = make(map[string]*websocket.Conn)
	d.eventChan = make(chan EventMessage, 2)
}

func NewStudioPointer() (d *Studio) {
	d = &Studio{}
	d.ID = GetXidString()
	d.initStudioValue()
	return
}

func NewStudioPointerWithName(name string, day int) (d *Studio) {
	d = NewStudioPointer()
	d.Name = name
	d.State = Using
	d.AtExpired = d.AtCreated.Add(time.Duration(day) * 24 * time.Hour)
	return
}

func (d *Studio) isEventState(state State) bool {
	d.Lock()
	defer d.Unlock()
	return d.EventState == state
}

func (d *Studio) setEventState(state State) {
	d.Lock()
	defer d.Unlock()
	d.EventState = state
}

// ---------------------------------------------------------------------------------
func (d *Studio) countNumbersByObjectState(object string, state State) (err error) {
	switch object {
	case "session":
		d.SessionGate.RLock()
		defer d.SessionGate.RUnlock()
		d.NumSessions[1] = 0
		for _, v := range d.Sessions {
			if v.State == state {
				d.NumSessions[1]++
			}
		}
	case "channel":
		d.ChannelGate.RLock()
		defer d.ChannelGate.RUnlock()
		d.NumChannels[1] = 0
		for _, v := range d.Channels {
			if v.State == Using {
				d.NumChannels[1]++
			}
		}
	case "group":
		d.GroupGate.RLock()
		defer d.GroupGate.RUnlock()
		d.NumGroups[1] = 0
		for _, v := range d.Groups {
			if v.State == Using {
				d.NumGroups[1]++
			}
		}
	case "bridge":
		d.BridgeGate.RLock()
		defer d.BridgeGate.RUnlock()
		d.NumBridges[1] = 0
		for _, v := range d.Bridges {
			if v.State == Using {
				d.NumBridges[1]++
			}
		}
	case "worker":
		d.WorkerGate.RLock()
		defer d.WorkerGate.RUnlock()
		d.NumWorkers[1] = 0
		for _, v := range d.Workers {
			if v.State == Using {
				d.NumWorkers[1]++
			}
		}
	case "punch":
		d.PunchGate.RLock()
		defer d.PunchGate.RUnlock()
		d.NumPunches[1] = 0
		for _, v := range d.Punches {
			if v.State == Using {
				d.NumPunches[1]++
			}
		}
	default:
		err = fmt.Errorf("not support object: %s", object)
		return
	}
	return
}

func (d *Studio) countItemsUsing() {
	d.AtUsed = time.Now()

	d.NumSessions[0] = len(d.Sessions)
	d.NumChannels[0] = len(d.Channels)
	d.NumGroups[0] = len(d.Groups)
	d.NumBridges[0] = len(d.Bridges)
	d.NumWorkers[0] = len(d.Workers)
	d.NumPunches[0] = len(d.Punches)

	d.countNumbersByObjectState("session", Using)
	d.countNumbersByObjectState("channel", Using)
	d.countNumbersByObjectState("group", Using)
	d.countNumbersByObjectState("bridge", Using)
	d.countNumbersByObjectState("worker", Using)
	d.countNumbersByObjectState("punch", Using)
}

// ---------------------------------------------------------------------------------
// Studio: Session
// ---------------------------------------------------------------------------------
func (d *Studio) listSessionsByChannelIDSessionName(cid, sname string) (ses []*Session) {
	d.SessionGate.RLock()
	defer d.SessionGate.RUnlock()

	for _, v := range d.Sessions {
		if v.ChannelID == cid {
			if sname != "" {
				if !strings.Contains(v.Name, sname) {
					continue
				}
			}
			ses = append(ses, v)
		}
	}
	return
}

func (d *Studio) findSessionByChannelIDSessionID(cid, sid string) (s *Session) {
	d.SessionGate.RLock()
	defer d.SessionGate.RUnlock()

	for _, v := range d.Sessions {
		if v.ChannelID == cid && v.ID == sid {
			return v
		}
	}
	return
}

// ---------------------------------------------------------------------------------
func (d *Studio) countSessionsByState(state string) (count int) {
	d.SessionGate.RLock()
	defer d.SessionGate.RUnlock()

	if state == "" {
		return len(d.Sessions)
	}
	for _, v := range d.Sessions {
		if v.State.String() == state {
			count++
		}
	}
	return
}

func (d *Studio) countSessionsByChannelID(cid string) (count int) {
	d.SessionGate.RLock()
	defer d.SessionGate.RUnlock()

	for _, v := range d.Sessions {
		if v.ChannelID == cid {
			count++
		}
	}
	return
}

func (d *Studio) countSessionsByBridgeID(bid string) (count int) {
	d.SessionGate.RLock()
	defer d.SessionGate.RUnlock()

	for _, v := range d.Sessions {
		if v.BridgeID == bid {
			count++
		}
	}
	return
}

// ---------------------------------------------------------------------------------
func (d *Studio) findPublisherByResource(cid, source, track string) (s *Session) {
	d.SessionGate.RLock()
	defer d.SessionGate.RUnlock()

	for _, v := range d.Sessions {
		if (v.ChannelID == cid && v.SourceID == source && v.TrackID == track) &&
			(strings.Contains(v.Name, "/pub") ||
				strings.Contains(v.Name, "/seb") ||
				strings.Contains(v.Name, "/meb")) {
			return v
		}
	}
	return
}

func (d *Studio) addNewSessionWithNameRequest(name, reqid string) (r *Session) {
	p := NewSessionPointerWithNameRequest(name, reqid)
	r = d.addSession(p)
	return
}

func (d *Studio) addNewSessionWithName(name string) (r *Session) {
	p := NewSessionPointerWithName(name)
	r = d.addSession(p)
	return
}

func (d *Studio) deleteSessionWithClose(p *Session) (err error) {
	p.State = Idle
	p.close()
	d.deleteSession(p)
	return
}

// ..............................................................
// Concurrency control : Session
// ..............................................................
func (d *Studio) findSessionByBridgeID(bid string) (r *Session) {
	d.SessionGate.RLock()
	defer d.SessionGate.RUnlock()

	for _, v := range d.Sessions {
		if v.BridgeID == bid {
			return v
		}
	}
	return
}

func (d *Studio) findSessionByID(pid string) (r *Session) {
	d.SessionGate.RLock()
	defer d.SessionGate.RUnlock()

	r = d.Sessions[pid]
	return
}

// func (d *Studio) closeSessionByID(pid string) (r *Session) {
// 	d.SessionGate.RLock()
// 	defer d.SessionGate.RUnlock()

// 	r = d.Sessions[pid]
// 	if r != nil {
// 		r.State = StateIdle
// 	}
// 	return
// }

func (d *Studio) addSession(p *Session) (r *Session) {
	d.SessionGate.Lock()
	defer d.SessionGate.Unlock()

	d.Sessions[p.ID] = p
	r = d.Sessions[p.ID]
	return
}

func (d *Studio) deleteSession(p *Session) {
	d.SessionGate.Lock()
	defer d.SessionGate.Unlock()

	delete(d.Sessions, p.ID)
}

// ---------------------------------------------------------------------------------
// Studio: Channel
// ---------------------------------------------------------------------------------
func (d *Studio) countChannelsByState(state string) (count int) {
	d.ChannelGate.RLock()
	defer d.ChannelGate.RUnlock()

	if state == "" {
		return len(d.Channels)
	}
	for _, c := range d.Channels {
		if c.State.String() == state {
			count++
		}
	}
	return count
}

// ---------------------------------------------------------------------------------
func (d *Studio) cleanChannels() (err error) {
	d.ChannelGate.Lock()
	defer d.ChannelGate.Unlock()

	for _, v := range d.Channels {
		if !v.isState(Idle) || !v.isEventState(Idle) {
			continue
		}
		switch v.Style {
		case "static":
			if len(v.Sources) == 0 { // already cleared, fixed v0.1.1.6
				continue
			}
			if time.Now().After(v.AtUsed.Add(time.Hour)) {
				v.purgeAllSourceTracks()
				log.Println("purged channel resource:", v.ID, v.Name, v.Style)
			}
		default: // instant, dynamic
			if time.Now().After(v.AtExpired) {
				delete(d.Channels, v.ID)
				log.Println("expired channel:", v.ID, v.Name, v.Style)
			}
		}
	}
	return
}

// ---------------------------------------------------------------------------------
func (d *Studio) addChannelInNameStyleOptions(name, style, key, record, trans string) (r *Channel) {
	p := NewChannelPointer()
	if name != "" {
		p.Name = name
	} else {
		p.Name = p.ID
	}
	if key != "" {
		p.StreamKey = key
	}
	p.Style = style
	switch p.Style {
	case "static":
	case "dynamic":
		p.AtExpired = time.Now().Add(TIME_EXPIRE_DYNAMIC)
	case "instant":
		p.AtExpired = time.Now().Add(TIME_EXPIRE_INSTANT)
	default:
		log.Println("invalid channel style:", style)
		return
	}
	switch record {
	case "on":
		p.RecordAuto = true
	case "off":
		p.RecordAuto = false
	}
	switch trans {
	case "on":
		p.TransAuto = true
	case "off":
		p.TransAuto = false
	}
	r = d.addChannel(p)
	return
}

// ---------------------------------------------------------------------------------
func (d *Studio) setChannelByQueryOption(qo QueryOption) (r *Channel) {
	channel := qo.Channel
	if !IsXidString(channel.ID) {
		channel.Style = channel.ID // Style <= ID
		switch channel.Style {
		case "instant", "dynamic":
			r = pStudio.findChannelByNameStyle(channel.Name, channel.Style)
			if r == nil {
				r = pStudio.addChannelInNameStyleOptions(channel.Name, channel.Style, channel.Key, channel.Record, channel.Trans)
				if r == nil {
					log.Println("channel alloc error:", channel.Name, channel.Style)
					return
				}
			}
		case "static":
			r = pStudio.findChannelByNameStyle(channel.Name, channel.Style)
			if r == nil {
				log.Println("not found channel:", channel.Name, channel.Style)
				return
			}
		default:
			log.Println("invalid channel info:", channel.Name, channel.Style)
			return
		}
	}
	return
}

// ..............................................................
// Concurrency control : Channel
// ..............................................................
func (d *Studio) getChannelByIDState(pid string, state State) (r *Channel) {
	d.ChannelGate.RLock()
	defer d.ChannelGate.RUnlock()

	r = d.Channels[pid]
	if r == nil || r.State != state {
		return nil
	}
	return
}

func (d *Studio) setChannelByIDState(pid string, state State) (r *Channel) {
	d.ChannelGate.Lock()
	defer d.ChannelGate.Unlock()

	r = d.Channels[pid]
	if r == nil {
		return nil
	}
	r.State = state
	if r.State == Idle { // fixed at v0.1.2.2 (2022/09/13)
		r.reset()
	}
	return
}

func (d *Studio) findChannelByName(name string) (r *Channel) {
	d.ChannelGate.RLock()
	defer d.ChannelGate.RUnlock()

	for _, ch := range d.Channels {
		if ch.Name == name {
			r = ch
			return
		}
	}
	return
}

func (d *Studio) findChannelIDByName(name string) (id string) {
	d.ChannelGate.RLock()
	defer d.ChannelGate.RUnlock()

	for _, ch := range d.Channels {
		if ch.Name == name {
			id = ch.ID
			return
		}
	}
	return
}

func (d *Studio) findChannelByNameStyle(name, style string) (r *Channel) {
	d.ChannelGate.RLock()
	defer d.ChannelGate.RUnlock()

	for _, ch := range d.Channels {
		if ch.Style == style && ch.Name == name {
			r = ch
			return
		}
	}
	return
}

func (d *Studio) findChannelByIDState(pid string, state State) (r *Channel) {
	d.ChannelGate.RLock()
	defer d.ChannelGate.RUnlock()

	r = d.Channels[pid]
	if r == nil || r.State != state {
		return nil
	}
	return
}

func (d *Studio) findChannelByID(pid string) (r *Channel) {
	d.ChannelGate.RLock()
	defer d.ChannelGate.RUnlock()

	r = d.Channels[pid]
	return
}

func (d *Studio) addChannel(p *Channel) (r *Channel) {
	d.ChannelGate.Lock()
	defer d.ChannelGate.Unlock()

	d.Channels[p.ID] = p
	r = d.Channels[p.ID]
	return
}

func (d *Studio) deleteChannel(p *Channel) {
	d.ChannelGate.Lock()
	defer d.ChannelGate.Unlock()

	p.close()
	delete(d.Channels, p.ID)
}

// ---------------------------------------------------------------------------------
// Studio: Worker
// ---------------------------------------------------------------------------------
func (d *Studio) addNewWorkerWithParams(name, chid, style string) (r *Worker) {
	p := NewWorkerPointerWithParams(name, chid, style)
	r = d.addWorker(p)
	return
}

// func (d *Studio) deleteWorkerWithClose(p *Worker) (err error) {
// 	d.deleteWorker(p)
// 	return
// }

// ..............................................................
// Concurrency control : Worker
// ..............................................................
func (d *Studio) findWorkerByID(pid string) (r *Worker) {
	d.WorkerGate.RLock()
	defer d.WorkerGate.RUnlock()

	r = d.Workers[pid]
	return
}

func (d *Studio) addWorker(p *Worker) (r *Worker) {
	d.WorkerGate.Lock()
	defer d.WorkerGate.Unlock()

	d.Workers[p.ID] = p
	r = d.Workers[p.ID]
	return
}

func (d *Studio) deleteWorker(p *Worker) {
	d.WorkerGate.Lock()
	defer d.WorkerGate.Unlock()

	delete(d.Workers, p.ID)
}

// ---------------------------------------------------------------------------------
// Studio: Punch for hole punching information
// ---------------------------------------------------------------------------------
func (d *Studio) cleanPunches() (err error) {
	d.PunchGate.Lock()
	defer d.PunchGate.Unlock()

	for _, v := range d.Punches {
		if time.Now().After(v.AtExpired) {
			delete(d.Punches, v.ID)
			log.Println("expired punch:", v.ID, v.Name, v.Role, v.Addr)
		}
	}
	return
}

// func (d *Studio) addNewPunchWithParams(name, chid string) (r *Punch) {
// 	p := NewPunchPointerWithParams(name, chid)
// 	r = d.addPunch(p)
// 	return
// }

// func (d *Studio) deletePunchWithClose(p *Punch) (err error) {
// 	d.deletePunch(p)
// 	return
// }

// ..............................................................
// Concurrency control : Punch
// ..............................................................
func (d *Studio) findPunchByName(name string) (r *Punch) {
	d.PunchGate.RLock()
	defer d.PunchGate.RUnlock()

	for _, ph := range d.Punches {
		if ph.Name == name {
			r = ph
			return
		}
	}
	return
}

func (d *Studio) findPunchByID(pid string) (r *Punch) {
	d.PunchGate.RLock()
	defer d.PunchGate.RUnlock()

	r = d.Punches[pid]
	return
}

func (d *Studio) addPunch(p *Punch) (r *Punch) {
	d.PunchGate.Lock()
	defer d.PunchGate.Unlock()

	d.Punches[p.ID] = p
	r = d.Punches[p.ID]
	return
}

func (d *Studio) deletePunch(p *Punch) {
	d.PunchGate.Lock()
	defer d.PunchGate.Unlock()

	delete(d.Punches, p.ID)
}

// ---------------------------------------------------------------------------------
// Studio: Event
// ---------------------------------------------------------------------------------
func (d *Studio) pushEvent(name, path, data string) {
	em := EventMessage{Type: "event", ID: GetXidString(),
		Name: name, Path: path, Data: data, AtCreated: time.Now()}
	d.eventChan <- em
}

func (d *Studio) addEventer(id string, ws *websocket.Conn) {
	d.EventerGate.Lock()
	defer d.EventerGate.Unlock()
	d.Eventers[id] = ws
}

func (d *Studio) deleteEventer(id string) {
	d.EventerGate.Lock()
	defer d.EventerGate.Unlock()
	delete(d.Eventers, id)
}

//=================================================================================
