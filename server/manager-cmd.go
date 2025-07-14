// =================================================================================
// Filename: manager-cmd.go
// Function: command processing for manager api
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2020-2023, 2025
// =================================================================================
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// --------------------------------------------------------------------------------
type Command struct {
	// Common
	Type string `json:"type,omitempty"`
	Time string `json:"time,omitempty"` // time.Time in Common
	// Below are used in CLI
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	State  string `json:"state,omitempty"` // State : State string in Common
	Style  string `json:"style,omitempty"` // Style
	Op     string `json:"op"`              // Operation
	Obj    string `json:"obj,omitempty"`   // Object
	Opt    string `json:"opt,omitempty"`   // Option
	Value  string `json:"value,omitempty"`
	Format string `json:"format,omitempty"`
	Key    string `json:"key,omitempty"` // Key for Manager
	// Below are not used in CLI
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
	Base   string `json:"base,omitempty"`
	Data   string `json:"data,omitempty"`
	url    string `json:"-"`
}

func NewCommandPointer() (cmd *Command) {
	cmd = &Command{}
	cmd.Type = "command"
	cmd.Time = time.Now().String()
	return
}

func (d *Command) String() (str string) {
	str += fmt.Sprintf("[%s] op:%s, obj:%s, id:%s, name:%s, state:%s, format:%s, opt:%s, value:%s",
		d.Type, d.Op, d.Obj, d.ID, d.Name, d.State, d.Format, d.Opt, d.Value)
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) parseQuery(r *http.Request) (err error) {
	log.Println("i.parseQuery:", r.URL)

	query := r.URL.Query()

	cmd.Op = query.Get("op")
	if cmd.Op == "" {
		err = fmt.Errorf("command op is required")
		return
	}
	cmd.Obj = query.Get("obj")

	cmd.ID = query.Get("id")
	cmd.Opt = query.Get("opt")
	cmd.Key = query.Get("key")
	cmd.Name = query.Get("name")
	cmd.Value = query.Get("value")
	cmd.State = query.Get("state")   // idle, using, close, block
	cmd.Style = query.Get("style")   // static, instant, dynamic
	cmd.Format = query.Get("format") // text, json, base64, form, ...
	if cmd.Format == "" {
		cmd.Format = "text"
	}
	if cmd.ID != "" && cmd.Name != "" {
		cmd.ID = pStudio.findChannelIDByName(cmd.Name)
		if cmd.ID == "" {
			err = fmt.Errorf("not found channel: %s", cmd.Name)
			return
		}
	}
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) execManager() (str string, err error) {
	// log.Println("i.execManager:", cmd.Op)

	switch cmd.Op {
	case "ping":
		str = "Pong!"
		if cmd.Format == "json" {
			str = fmt.Sprintf("{ \"result\": \"%s\" }", str)
		}
	case "show":
		switch cmd.Obj {
		case "session":
			if cmd.ID == "" {
				str, err = cmd.showSessions()
			} else {
				str, err = cmd.infoSession()
			}
		case "channel":
			if cmd.ID == "" {
				str, err = cmd.showChannels()
			} else {
				str, err = cmd.infoChannel()
			}
		case "bridge":
			if cmd.ID == "" {
				str, err = cmd.showBridges()
			} else {
				str, err = cmd.infoBridge()
			}
		case "worker":
			if cmd.ID == "" {
				str, err = cmd.showWorkers()
			} else {
				str, err = cmd.infoWorker()
			}
		case "studio":
			str, err = cmd.showStudio()
		case "config":
			str, err = cmd.showConfig()
		case "system":
			str, err = cmd.showSystem()
		case "dir":
			str, err = cmd.showDir()
		default:
			err = fmt.Errorf("invalid obj %s for %s", cmd.Obj, cmd.Op)
			return
		}
	case "set":
		switch cmd.Obj {
		case "session":
			str, err = cmd.setSession()
		case "channel":
			str, err = cmd.setChannel()
		case "bridge":
			str, err = cmd.setBridge()
		case "worker":
			str, err = cmd.setWorker()
		case "config":
			str, err = cmd.setConfig()
		default:
			err = fmt.Errorf("invalid obj %s for %s", cmd.Obj, cmd.Op)
			return
		}
	case "run":
		switch cmd.Obj {
		case "bridge":
			str, err = cmd.runBridge()
		case "shell":
			str, err = cmd.runShell()
		default:
			err = fmt.Errorf("invalid obj %s for %s", cmd.Obj, cmd.Op)
			return
		}
	case "add":
		switch cmd.Obj {
		case "channel":
			str, err = cmd.addChannel()
		case "bridge":
			str, err = cmd.addBridge()
		default:
			err = fmt.Errorf("invalid obj %s for %s", cmd.Obj, cmd.Op)
			return
		}
	case "delete":
		switch cmd.Obj {
		case "channel":
			str, err = cmd.deleteChannel()
		case "bridge":
			str, err = cmd.deleteBridge()
		default:
			err = fmt.Errorf("invalid obj %s for %s", cmd.Obj, cmd.Op)
			return
		}
	case "clean":
		switch cmd.Obj {
		case "channel":
			str, err = cmd.cleanChannels()
		default:
			err = fmt.Errorf("invalid obj %s for %s", cmd.Obj, cmd.Op)
			return
		}
	case "load":
		switch cmd.Obj {
		case "channel":
			str, err = cmd.loadChannels()
		default:
			err = fmt.Errorf("invalid obj %s for %s", cmd.Obj, cmd.Op)
			return
		}
	case "save":
		switch cmd.Obj {
		case "channel":
			str, err = cmd.saveChannels()
		default:
			err = fmt.Errorf("invalid obj %s for %s", cmd.Obj, cmd.Op)
			return
		}
	case "check":
		switch cmd.Obj {
		case "channel":
			str, err = cmd.checkChannelResource()
		default:
			err = fmt.Errorf("invalid obj %s for %s", cmd.Obj, cmd.Op)
			return
		}
	default:
		err = fmt.Errorf("not supported op %s", cmd.Op)
	}
	return
}

// ---------------------------------------------------------------------------------
func (cmd *Command) checkManagerPermission() (err error) {
	log.Println("i.checkManagerPermission:", cmd.Op, cmd.Obj)

	if mConfig.KeyManager != "" {
		switch cmd.Op {
		case "show":
			if cmd.Obj == "config" && mConfig.KeyManager != cmd.Key {
				err = fmt.Errorf("invalid manager key")
				return
			}
		default:
			if mConfig.KeyManager != cmd.Key {
				err = fmt.Errorf("invalid manager key")
				return
			}
		}
	}
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) showSessions() (str string, err error) {
	log.Println("i.showSessions:", cmd.ID, cmd.State, cmd.Value)
	pStudio.SessionGate.RLock()
	defer pStudio.SessionGate.RUnlock()

	var commons []Common
	for _, v := range pStudio.Sessions {
		if cmd.State != "" {
			switch cmd.State {
			case "block":
				if !v.Common.Blocked {
					continue
				}
			case "idle", "using":
				if cmd.State != v.State.String() {
					continue
				}
			}
		}
		if cmd.Value != "" {
			if !strings.Contains(v.Name, cmd.Value) {
				continue
			}
		}
		commons = append(commons, v.Common)
	}
	str = StringSortCommonListByFormat(commons, len(pStudio.Sessions), cmd.Format)
	return
}

func (cmd *Command) infoSession() (str string, err error) {
	log.Println("i.infoSession:", cmd.ID)
	p := pStudio.findSessionByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found session: %s", cmd.ID)
		return
	}
	if p.req != nil {
		p.Desc = p.req.RequestURI
		p.RemoteAddr = p.req.RemoteAddr
	}
	p.AtUsed = time.Now()
	return FormatItem(p, cmd.Format)
}

func (cmd *Command) setSession() (str string, err error) {
	log.Println("setSession:", cmd.ID, cmd.State)
	p := pStudio.findSessionByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found session: %s", cmd.ID)
		return
	}
	switch cmd.State {
	case "close":
		p.State = Idle
	}
	return FormatItem(p, cmd.Format)
}

// --------------------------------------------------------------------------------
func (cmd *Command) infoChannelTracks() (str string, err error) {
	log.Println("i.infoChannelTracks:", cmd.ID, cmd.Opt)
	p := pStudio.findChannelByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found channel: %s", cmd.ID)
		return
	}
	if cmd.Format == "json" {
		data, _ := json.MarshalIndent(p.Sources, "", "   ")
		str = string(data)
	} else {
		str = p.ListResources("", cmd.Value)
	}
	return
}

func (cmd *Command) infoChannelSources() (str string, err error) {
	log.Println("i.infoChannelSources:", cmd.ID, cmd.Opt)
	p := pStudio.findChannelByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found channel: %s", cmd.ID)
		return
	}
	if cmd.Format == "json" {
		data, _ := json.MarshalIndent(p.Sources, "", "   ")
		str = string(data)
	} else {
		str = p.ListResources(cmd.Value, "")
	}
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) infoChannelActors(actor string) (str string, err error) {
	log.Println("i.infoChannelActors:", cmd.ID, cmd.Opt)

	var commons []Common
	for _, v := range pStudio.Sessions {
		if strings.Contains(v.Name, actor) && v.ChannelID == cmd.ID {
			commons = append(commons, v.Common)
		}
	}
	str = StringSortCommonListByFormat(commons, len(pStudio.Sessions), cmd.Format)
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) showChannels() (str string, err error) {
	log.Println("i.showChannels:", cmd.ID, cmd.State, cmd.Style, cmd.Value)
	pStudio.ChannelGate.RLock()
	defer pStudio.ChannelGate.RUnlock()

	var commons []Common
	for _, v := range pStudio.Channels {
		switch cmd.State {
		case "idle", "using":
			if cmd.State != v.State.String() {
				continue
			}
		}
		switch cmd.Style {
		case "static", "dynamic", "instant":
			if cmd.Style != v.Style {
				continue
			}
		}
		switch cmd.Opt {
		case "block":
			if cmd.State == "off" {
				if v.Common.Blocked {
					continue
				}
			} else {
				if !v.Common.Blocked {
					continue
				}
			}
		case "name":
			if cmd.Value != "" {
				if !strings.Contains(v.Name, cmd.Value) {
					continue
				}
			}
		case "key":
			if cmd.State == "on" {
				if v.StreamKey == "" {
					continue
				}
			}
		}
		if cmd.Value != "" {
			if !strings.Contains(v.Name, cmd.Value) {
				continue
			}
		}
		commons = append(commons, v.Common)
	}
	str = StringSortCommonListByFormat(commons, len(pStudio.Channels), cmd.Format)
	return
}

func (cmd *Command) infoChannel() (str string, err error) {
	log.Println("infoChannel:", cmd.ID)
	p := pStudio.findChannelByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found channel: %s", cmd.ID)
		return
	}
	switch cmd.Opt {
	case "track":
		str, err = cmd.infoChannelTracks()
	case "source":
		str, err = cmd.infoChannelSources()
	case "actor":
		str, err = cmd.infoChannelActors(cmd.Value) // ctl, pub, sub
	default:
		str, err = FormatItem(p, cmd.Format)
	}
	return
}

// func (cmd *Command) setChannelByNameStyleKey(name, style, key string) (str string, err error) {
// 	p := pStudio.setChannelByNameStyleKey(name, style, key)
// 	return FormatItem(p, cmd.Format)
// }

func (cmd *Command) setChannel() (str string, err error) {
	log.Println("i.setChannel:", cmd.ID, cmd.State, cmd.Opt)
	p := pStudio.findChannelByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found channel: %s", cmd.ID)
		return
	}
	switch cmd.Opt {
	case "state":
		if cmd.State == "close" {
			if p.isState(Using) {
				p.State = Idle
			}
		}
	case "block":
		if cmd.State == "on" {
			p.Blocked = true
		} else if cmd.State == "off" {
			p.Blocked = false
		}
	case "key":
		if cmd.State == "off" {
			p.StreamKey = ""
		} else if cmd.Value != "" {
			p.StreamKey = cmd.Value
		}
	case "name":
		if cmd.Value != "" {
			p.Name = cmd.Value
		} else {
			err = fmt.Errorf("no name for channel: %s, %s, %s", cmd.ID, cmd.Opt, cmd.Value)
			return
		}
	case "style":
		p.Style = cmd.Style
		switch cmd.Style {
		case "static":
			p.AtExpired = time.Time{}
		case "instant":
			p.AtExpired = time.Now().Add(TIME_EXPIRE_INSTANT)
		case "dynamic":
		default:
			err = fmt.Errorf("invalid style for channel: %s, %s, %s", cmd.ID, cmd.Opt, cmd.Value)
			return
		}
	default:
		err = fmt.Errorf("%s command unknown option: %s", cmd.Op, cmd.Opt)
		return
	}
	p.AtUpdated = time.Now()
	cmd.saveChannels()
	return FormatItem(p, cmd.Format)
}

func (cmd *Command) addChannel() (str string, err error) {
	log.Println("i.addChannel:", cmd.ID, cmd.Value)
	p := NewChannelPointer()
	if IsXidString(cmd.ID) {
		if pStudio.findChannelByID(cmd.ID) != nil {
			err = fmt.Errorf("already registered channel id: %s", cmd.ID)
			return
		}
		p.ID = cmd.ID
	}
	if cmd.Value != "" {
		p.Name = cmd.Value
	}
	switch cmd.Style {
	case "":
		p.Style = "static"
	case "static", "dynamic":
		p.Style = cmd.Style
	case "instant":
		p.Style = cmd.Style
		p.AtExpired = time.Now().Add(TIME_EXPIRE_INSTANT)
	default:
		err = fmt.Errorf("invalid channel style: %s", cmd.Style)
		return
	}
	switch cmd.Opt {
	case "block":
		p.Blocked = true
	}
	if pStudio.findChannelByName(p.Name) != nil {
		err = fmt.Errorf("already registered channel name: %s", p.Name)
		return
	}
	pStudio.addChannel(p)
	cmd.saveChannels()
	return FormatItem(p, cmd.Format)
}

func (cmd *Command) deleteChannel() (str string, err error) {
	log.Println("i.deleteChannel:", cmd.ID)
	p := pStudio.findChannelByID(cmd.ID)
	if p == nil || p.State != Idle {
		err = fmt.Errorf("invalid channel or state: %s", cmd.ID)
		return
	}
	str, err = FormatItem(p, cmd.Format)
	pStudio.deleteChannel(p)
	cmd.saveChannels()
	return
}

// --------------------------------------------------------------------------------
type Resource struct {
	Path  string `json:"path,omitempty"`
	State string `json:"state,omitempty"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

func (d *Resource) String() (str string) {
	str += "[resource] "
	if d.Path != "" {
		str += fmt.Sprintf("path: %s, ", d.Path)
	}
	if d.State != "" {
		str += fmt.Sprintf("state: %s, ", d.State)
	}
	if d.Name != "" {
		str += fmt.Sprintf("name: %s, ", d.Name)
	}
	if d.Value != "" {
		str += fmt.Sprintf("value: %s, ", d.Value)
	}
	return str
}

// --------------------------------------------------------------------------------
func (cmd *Command) checkChannelResource() (str string, err error) {
	log.Println("i.checkChannelResource:", cmd.ID, cmd.Value)
	p := pStudio.findChannelByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found channel: %s", cmd.ID)
		return
	}

	var resource = &Resource{}

	switch cmd.Opt {
	case "block":
		resource.Name = cmd.Opt
		resource.Value = fmt.Sprintf("%v", p.Blocked)
	case "key":
		resource.Name = cmd.Opt
		if p.StreamKey != "" {
			resource.Value = "*****"
		} else {
			resource.Value = "none"
		}
	case "path":
		toks := strings.Split(cmd.Value, "/")
		if len(toks) != 2 {
			err = fmt.Errorf("invalid resource path: %s/%s", cmd.ID, cmd.Value)
			return
		}

		resource.Path = fmt.Sprintf("%s/%s/%s", cmd.ID, toks[0], toks[1])
		resource.State = "idle"

		owner := pStudio.findPublisherByResource(cmd.ID, toks[0], toks[1])
		if owner != nil {
			resource.State = "using"
		}
	default:
		err = fmt.Errorf("invalid resource option: %s", cmd.Opt)
		return
	}
	str, err = FormatItem(resource, cmd.Format)
	log.Println(resource)
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) cleanChannels() (str string, err error) {
	log.Println("i.cleanChannels:", cmd.Op)
	err = pStudio.cleanChannels()
	return
}

func (cmd *Command) loadChannels() (str string, err error) {
	log.Println("i.loadChannels:", cmd.Op)
	err = pStudio.readObjectFileInArray("channel", "conf/channels.json")
	return
}

func (cmd *Command) saveChannels() (str string, err error) {
	log.Println("i.saveChannels:", cmd.Op)
	err = os.Rename("conf/channels.json", "conf/channels.json.bak")
	if err != nil {
		log.Println(err)
		return
	}
	err = pStudio.writeObjectFileInArray("channel", "conf/channels.json")
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) showBridges() (str string, err error) {
	log.Println("i.showBridges:", cmd.ID)
	pStudio.BridgeGate.RLock()
	defer pStudio.BridgeGate.RUnlock()

	var commons []Common
	for _, v := range pStudio.Bridges {
		if cmd.State != "" && cmd.State != v.Common.State.String() {
			continue
		} else {
			commons = append(commons, v.Common)
		}
	}
	str = StringSortCommonListByFormat(commons, len(pStudio.Bridges), cmd.Format)
	return
}

func (cmd *Command) infoBridge() (str string, err error) {
	log.Println("infoBridge:", cmd.ID)
	p := pStudio.findBridgeByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found bridge: %s", cmd.ID)
		return
	}
	switch cmd.Opt {
	default:
		str, err = FormatItem(p, cmd.Format)
	}
	return
}

func (cmd *Command) setBridge() (str string, err error) {
	log.Println("i.setBridge:", cmd.ID, cmd.State, cmd.Opt)
	p := pStudio.findBridgeByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found bridge: %s", cmd.ID)
		return
	}
	switch cmd.Opt {
	case "state":
		if cmd.State == "close" {
			if p.isState(Using) {
				p.State = Idle
			}
		}
	case "block":
		if cmd.State == "on" {
			p.Blocked = true
		} else if cmd.State == "off" {
			p.Blocked = false
		}
	case "auto":
		if cmd.State == "on" {
			p.Attr = "auto"
		} else if cmd.State == "off" {
			p.Attr = "none"
		}
	default:
		err = fmt.Errorf("invalid bridge option: %s", cmd.Opt)
	}
	return
}

func (cmd *Command) runBridge() (str string, err error) {
	log.Println("i.runBridge:", cmd.ID, cmd.State)
	p := pStudio.findBridgeByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found bridge: %s", cmd.ID)
		return
	}
	switch cmd.State {
	case "start":
		if p.isState(Using) || p.Blocked {
			err = fmt.Errorf("invalid bridge state: %s or block: %v", p.State, p.Blocked)
			return
		}
		err = pStudio.startBridge(p)
	case "stop":
		if p.isState(Idle) {
			err = fmt.Errorf("invalid bridge state: %s", p.State)
			return
		}
		err = pStudio.stopBridge(p)
	default:
		err = fmt.Errorf("command invalid action: %s [start|stop]", cmd.State)
	}
	return
}

func (cmd *Command) addBridge() (str string, err error) {
	log.Println("i.addBridge:", cmd.ID, cmd.Value)
	p := NewBridgePointer()
	if cmd.Value == "" {
		p.Name = p.ID
	} else {
		p.Name = cmd.Value
	}
	pStudio.addBridge(p)
	cmd.saveBridges()
	return FormatItem(p, cmd.Format)
}

func (cmd *Command) deleteBridge() (str string, err error) {
	log.Println("i.deleteBridge:", cmd.ID)
	p := pStudio.findBridgeByID(cmd.ID)
	if p == nil || p.State != Idle {
		err = fmt.Errorf("invalid bridge or state: %s", cmd.ID)
		return
	}
	str, err = FormatItem(p, cmd.Format)
	pStudio.deleteBridge(p)
	cmd.saveBridges()
	return
}

func (cmd *Command) loadBridges() (str string, err error) {
	log.Println("i.loadBridges:", cmd.Op)
	err = pStudio.readObjectFileInArray("bridge", "conf/bridges.json")
	return
}

func (cmd *Command) saveBridges() (str string, err error) {
	log.Println("i.saveBridges:", cmd.Op)
	err = os.Rename("conf/bridges.json", "conf/bridges.json.bak")
	if err != nil {
		log.Println(err)
		return
	}
	err = pStudio.writeObjectFileInArray("bridge", "conf/bridges.json")
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) showWorkers() (str string, err error) {
	log.Println("i.showWorkers:", cmd.ID)
	pStudio.WorkerGate.RLock()
	defer pStudio.WorkerGate.RUnlock()

	var commons []Common
	for _, v := range pStudio.Workers {
		if cmd.State != "" && cmd.State != v.Common.State.String() {
			continue
		} else {
			commons = append(commons, v.Common)
		}
	}
	str = StringSortCommonListByFormat(commons, len(pStudio.Workers), cmd.Format)
	return
}

func (cmd *Command) infoWorker() (str string, err error) {
	log.Println("infoWorker:", cmd.ID)
	p := pStudio.findWorkerByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found worker: %s", cmd.ID)
		return
	}
	switch cmd.Opt {
	default:
		str, err = FormatItem(p, cmd.Format)
	}
	return
}

func (cmd *Command) setWorker() (str string, err error) {
	log.Println("i.setWorker:", cmd.ID, cmd.State, cmd.Opt)
	p := pStudio.findWorkerByID(cmd.ID)
	if p == nil {
		err = fmt.Errorf("not found worker: %s", cmd.ID)
		return
	}
	switch cmd.Opt {
	case "state":
		switch cmd.State {
		case "close":
			if p.Style == "system" {
				err = fmt.Errorf("system worker not allowed to set: %s", cmd.State)
				return
			}
			if p.isState(Using) {
				p.State = Idle
			}
		default:
			err = fmt.Errorf("invalid worker state: %s [close]", cmd.State)
		}
	default:
		err = fmt.Errorf("invalid worker option: %s [state]", cmd.Opt)
	}
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) showStudio() (str string, err error) {
	log.Println("i.showStudio:", cmd.ID)
	pStudio.countItemsUsing()
	return FormatItem(pStudio, cmd.Format)
}

// --------------------------------------------------------------------------------
func (cmd *Command) showConfig() (str string, err error) {
	log.Println("i.showConfig:")
	// log.Println(mConfig.license)
	return FormatItem(mConfig, cmd.Format)
}

func (cmd *Command) setConfig() (str string, err error) {
	log.Println("i.setConfig:", cmd.ID, cmd.Opt)

	if cmd.ID != mConfig.ID {
		err = fmt.Errorf("invalid config ID: %s", cmd.ID)
		return
	}
	switch cmd.Opt {
	case "key":
		if cmd.State == "off" {
			mConfig.KeyManager = ""
		} else if cmd.Value != "" {
			mConfig.KeyManager = cmd.Value
		}
	default:
		err = fmt.Errorf("not support %s option: %s", cmd.Op, cmd.Opt)
		return
	}
	mConfig.AtUpdated = time.Now()
	// cmd.saveConfig()
	return FormatItem(mConfig, cmd.Format)
}

func (cmd *Command) saveConfig() (str string, err error) {
	log.Println("i.saveConfig:", cmd.Op)
	err = os.Rename("conf/moth.json", "conf/moth.json.bak")
	if err != nil {
		log.Println(err)
		return
	}
	err = pStudio.writeObjectFileInArray("config", "conf/moth.json")
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) showSystem() (str string, err error) {
	log.Println("i.showSystem:", cmd.Name)
	switch cmd.Opt {
	case "disk":
		disk, err := GetDiskUsage("/")
		if err != nil {
			return "", err
		}
		str, err = FormatItem(disk, cmd.Format)
	default:
		si := GetSystemInfo()
		str, err = FormatItem(si, cmd.Format)
	}
	return
}

// --------------------------------------------------------------------------------
func StringSortCommonListByFormat(commons []Common, total int, format string) (str string) {
	sort.Slice(commons, func(i, j int) bool {
		return commons[i].ID < commons[j].ID
	})
	if format == "json" {
		data, _ := json.MarshalIndent(commons, "", "   ")
		str = string(data)
	} else {
		for i := range commons {
			str += commons[i].String() + "\n"
		}
		str += fmt.Sprintf("Total: %d/%v", len(commons), total)
	}
	return
}

// --------------------------------------------------------------------------------
type Item struct {
	Dir  string
	Name string
	Size int64
}

func (cmd *Command) showDir() (str string, err error) {
	log.Println("showDir:", cmd.Value)
	if cmd.Value == "" {
		cmd.Value = "."
	}

	var Items []Item

	files, err := os.ReadDir(cmd.Value)
	if err != nil {
		err = fmt.Errorf("ReadDir: %s", err)
		return
	}

	for _, f := range files {
		fname := f.Name()
		if f.IsDir() {
			fname += "/"
		}
		finfo, _ := f.Info()
		item := Item{Dir: cmd.Value, Name: fname, Size: finfo.Size()}
		Items = append(Items, item)
	}

	if cmd.Format == "json" {
		data, _ := json.MarshalIndent(Items, "", "   ")
		str = string(data)
	} else {
		for _, item := range Items {
			str += fmt.Sprintf("%10v %s/%s \n", item.Size, item.Dir, item.Name)
		}
		str += fmt.Sprintf("Total: %v", len(Items))
	}
	return
}

// --------------------------------------------------------------------------------
func (cmd *Command) runShell() (str string, err error) {
	log.Println("runShell:", cmd.Value)

	toks := strings.Fields(cmd.Value)
	if len(toks) < 1 {
		err = fmt.Errorf("no shell command")
		return
	}

	out, err := exec.Command(toks[0], toks[1:]...).CombinedOutput()
	if err != nil {
		log.Println(err)
	}
	str = string(out)
	return
}

//=================================================================================
