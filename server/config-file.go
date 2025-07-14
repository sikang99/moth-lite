// =================================================================================
// Filename: config-file.go
// Function: handling for config files
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-2023
// =================================================================================
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"time"
)

// ---------------------------------------------------------------------------------
func (d *Studio) readObjectFileInMap(object, fname string) (err error) {
	log.Println("i.readObjectFileInMap:", object, fname)

	file, err := os.Open(fname)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Println(err)
		return
	}
	// log.Println(string(data))

	// CAUTION : ReadConfig -> Write lock
	switch object {
	case "channel":
		pStudio.ChannelGate.Lock()
		defer pStudio.ChannelGate.Unlock()

		err = json.Unmarshal(data, &pStudio.Channels)
		if err != nil {
			log.Println("[ERR]", err, fname)
			return
		}
		for _, v := range pStudio.Channels {
			v.newChannelValue()
		}
	case "group":
		pStudio.GroupGate.Lock()
		defer pStudio.GroupGate.Unlock()

		err = json.Unmarshal(data, &pStudio.Groups)
		if err != nil {
			log.Println(err)
			return
		}
		for _, v := range pStudio.Groups {
			v.setGroupValue()
		}
	default:
		err = fmt.Errorf("not supported object: %s", object)
	}
	return
}

// ---------------------------------------------------------------------------------
func (d *Studio) writeObjectFileInMap(object, fname string) (err error) {
	log.Println("i.writeObjectFileInMap:", object, fname)

	file, err := os.Create(fname)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer file.Close()

	// CAUTION : WriteConfig -> Read lock
	switch object {
	case "channel":
		pStudio.ChannelGate.RLock()
		defer pStudio.ChannelGate.RUnlock()

		data, _ := json.MarshalIndent(pStudio.Channels, "", "   ")
		_, err = file.Write(data)
		if err != nil {
			log.Println(err)
			return
		}
	case "group":
		pStudio.GroupGate.RLock()
		defer pStudio.GroupGate.RUnlock()

		data, _ := json.MarshalIndent(pStudio.Groups, "", "   ")
		_, err = file.Write(data)
		if err != nil {
			log.Println(err)
			return
		}
	default:
		err = fmt.Errorf("not supported object: %s", object)
	}
	// log.Println(string(data))
	return
}

// ---------------------------------------------------------------------------------
func (d *Studio) readObjectFileInArray(object, fname string) (err error) {
	log.Println("i.readObjectFileInArray:", object, fname)

	file, err := os.Open(fname)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Println(err)
		return
	}

	// CAUTION : ReadConfig -> Write lock
	switch object {
	case "channel":
		pStudio.ChannelGate.Lock()
		defer pStudio.ChannelGate.Unlock()

		var channels []*Channel
		err = json.Unmarshal(data, &channels)
		if err != nil {
			err = fmt.Errorf("json unmarshal: %s for %s", err, object)
			return
		}
		for _, v := range channels {
			v.setChannelValue()
			if v.ID == "" {
				v.ID = GetXidString()
			}
			if d.Channels[v.ID] != nil {
				log.Printf("[WARN] already registered %s id: %s", object, v.ID)
				continue
			}
			if v.AtCreated.IsZero() {
				v.AtCreated = time.Now()
			}
			d.Channels[v.ID] = v
		}
	case "group":
		pStudio.GroupGate.Lock()
		defer pStudio.GroupGate.Unlock()

		var groups []*Group
		err = json.Unmarshal(data, &groups)
		if err != nil {
			err = fmt.Errorf("json unmarshal: %s for %s", err, object)
			return
		}
		for _, v := range groups {
			v.setGroupValue()
			if v.ID == "" {
				v.ID = GetXidString()
			}
			if d.Groups[v.ID] != nil {
				log.Printf("[WARN] already registered %s id: %s", object, v.ID)
				continue
			}
			if v.AtCreated.IsZero() {
				v.AtCreated = time.Now()
			}
			d.Groups[v.ID] = v
		}
	case "bridge":
		pStudio.BridgeGate.Lock()
		defer pStudio.BridgeGate.Unlock()

		var bridges []*Bridge
		err = json.Unmarshal(data, &bridges)
		if err != nil {
			err = fmt.Errorf("json unmarshal: %s for %s", err, object)
			return
		}
		for _, v := range bridges {
			v.setBridgeValue()
			if v.ID == "" {
				v.ID = GetXidString()
			}
			if d.Bridges[v.ID] != nil {
				log.Printf("[WARN] already registered %s id: %s", object, v.ID)
				continue
			}
			if v.AtCreated.IsZero() {
				v.AtCreated = time.Now()
			}
			d.Bridges[v.ID] = v
		}
	default:
		err = fmt.Errorf("not supported object: %s", object)
	}
	return
}

// ---------------------------------------------------------------------------------
func (d *Studio) writeObjectFileInArray(object, fname string) (err error) {
	log.Println("i.writeObjectFileInArray:", object, fname)

	file, err := os.Create(fname)
	if err != nil {
		log.Println("[ERR]", err)
		return
	}
	defer file.Close()

	// CAUTION : WriteConfig -> Read lock
	switch object {
	case "channel":
		pStudio.ChannelGate.RLock()
		defer pStudio.ChannelGate.RUnlock()

		var channels []*Channel
		for _, v := range d.Channels {
			channels = append(channels, v)
		}

		sort.Slice(channels, func(i, j int) bool {
			return channels[i].ID < channels[j].ID
		})

		data, _ := json.MarshalIndent(channels, "", "   ")
		_, err = file.Write(data)
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
	case "group":
		pStudio.GroupGate.RLock()
		defer pStudio.GroupGate.RUnlock()

		var groups []*Group
		for _, v := range d.Groups {
			groups = append(groups, v)
		}

		sort.Slice(groups, func(i, j int) bool {
			return groups[i].ID < groups[j].ID
		})

		data, _ := json.MarshalIndent(groups, "", "   ")
		_, err = file.Write(data)
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
	case "bridge":
		pStudio.BridgeGate.RLock()
		defer pStudio.BridgeGate.RUnlock()

		var bridges []*Bridge
		for _, v := range d.Bridges {
			bridges = append(bridges, v)
		}

		sort.Slice(bridges, func(i, j int) bool {
			return bridges[i].ID < bridges[j].ID
		})

		data, _ := json.MarshalIndent(bridges, "", "   ")
		_, err = file.Write(data)
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
	case "config":
		data, _ := json.MarshalIndent(mConfig, "", "   ")
		_, err = file.Write(data)
		if err != nil {
			log.Println("[ERR]", err)
			return
		}
	default:
		err = fmt.Errorf("not supported object: %s", object)
	}
	// log.Println(string(data))
	return
}

//---------------------------------------------------------------------------------
// func (d *Studio) checkConfigChange() (err error) {
// 	watcher, err := fsnotify.NewWatcher()
// 	if err != nil {
// 		log.Println(err)
// 		return
// 	}

// 	err = watcher.Add("./conf/channels.json")
// 	if err != nil {
// 		log.Println(err)
// 		return
// 	}

// 	for {
// 		select {
// 		case event := <-watcher.Events:
// 			log.Println("event:", event)
// 			// reload channels
// 		}
// 	}
// }

//=================================================================================
