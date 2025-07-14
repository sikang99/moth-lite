// =================================================================================
// Filename: config-moth.go
// Function: config for moth server
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-2025
// =================================================================================
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------------
// type and global variables
type MothConfig struct {
	Common       `json:",inline"`
	TimeZone     string        `json:"time_zone,omitempty"`
	ProgramTitle string        `json:"program_title,omitempty"`
	ProgramBase  string        `json:"program_base,omitempty"`
	ExternalIP   string        `json:"external_ip,omitempty"`
	ServerURL    string        `json:"server_url,omitempty"`
	HostAddr     string        `json:"host_addr,omitempty"`
	PortPlain    int           `json:"port_plain"`
	PortSecure   int           `json:"port_secure"`
	TCPPlain     int           `json:"tcp_plain"`
	TCPSecure    int           `json:"tcp_secure"`
	QUICSecure   int           `json:"quic_secure"`
	RaknetPlain  int           `json:"raknet_plain"`
	KCPPlain     int           `json:"kcp_plain"`
	PEMPublic    string        `json:"pem_public"`
	PEMPrivate   string        `json:"pem_private"`
	TypeRun      string        `json:"type_run,omitempty"`
	TypeSub      string        `json:"type_sub,omitempty"`
	TypeObject   string        `json:"type_object,omitempty"`
	MediaName    string        `json:"media_name,omitempty"`
	IDChannel    string        `json:"id_channel,omitempty"`
	IDGroup      string        `json:"id_group,omitempty"`
	DirLog       string        `json:"dir_log,omitempty"`
	DirData      string        `json:"dir_data,omitempty"`
	DirRecord    string        `json:"dir_record,omitempty"`
	KeyManager   string        `json:"key_manager"`
	KeyLicense   string        `json:"key_license"`
	SecTicket    bool          `json:"sec_ticket"`
	SecHost      bool          `json:"sec_host"`
	HostCIDR     string        `json:"host_cidr"`
	CORSAllow    bool          `json:"cors_allow"`
	COPAllow     bool          `json:"cop_allow"`
	Owner        string        `json:"owner,omitempty"`
	NumCount     int64         `json:"num_count,omitempty"`
	NumTotal     int64         `json:"num_total,omitempty"`
	NumPubs      int           `json:"num_pubs,omitempty"`
	NumSubs      int           `json:"num_subs,omitempty"`
	RunningTime  time.Duration `json:"running_time,omitempty"`
	// --- internal variables
	license License
	sync.Mutex
}

func (d *MothConfig) initConfigValue() {
	d.Type = "config"
	d.Name = "moth"
	d.State = Using
	d.AtCreated = time.Now()
	d.AtUsed = d.AtCreated
	d.AtExpired = d.AtCreated.AddDate(0, 0, 10) // AddDate(year, month, date), 1 month for demo license
	d.TypeRun = "server"
	d.HostAddr = "localhost"
	d.PortPlain = 8276
	d.PortSecure = 8277
	d.TCPPlain = 8274
	d.TCPSecure = 8275
	d.QUICSecure = 0  // 0: disable, 0 > : enable, 18277
	d.RaknetPlain = 0 // 0: disable, 0 > : enable
	d.KCPPlain = 0    // 0: disable, 0 > : enable
	d.PEMPublic = "cert/cert.pem"
	d.PEMPrivate = "cert/key.pem"
	d.DirLog = "./log"
	d.DirData = "./data"
	d.DirRecord = "./data/record"
	d.CORSAllow = true
	d.ExternalIP = GetExternalIPString()
	d.NumPubs = 2 // default number of publishers (channels)
	d.NumSubs = 5 // default number of subscribers (sessions)
}

func NewConfigPointer() (d *MothConfig) {
	d = &MothConfig{}
	d.initConfigValue()
	return
}

func (d *MothConfig) String() (str string) {
	str = d.Common.String()
	str += fmt.Sprintf("\n\t[Title] %s on %s", d.ProgramTitle, d.ProgramBase)
	str += fmt.Sprintf("\n\t[Server] HTTP/S: %4d/%4d, External: %s, Addr: %s, URL: %s, TCP/S: %d/%d, QUIC: %d",
		d.PortPlain, d.PortSecure, d.ExternalIP, d.HostAddr, d.ServerURL, d.TCPPlain, d.TCPSecure, d.QUICSecure)
	str += fmt.Sprintf("\n\t[Cert] Public: %s, Private: %s", d.PEMPublic, d.PEMPrivate)
	str += fmt.Sprintf("\n\t[Sec] Key: %8s, Ticket: %v, Host: %v (%s), CORSAllow: %v",
		d.KeyManager, d.SecTicket, d.SecHost, d.HostCIDR, d.CORSAllow)
	str += fmt.Sprintf("\n\t[Dirs] Log: %s, Data: %s, Record: %s", d.DirLog, d.DirData, d.DirRecord)
	str += fmt.Sprintf("\n\t[Stats] Count: %v, Total: %v, Max In/Out: %d/%d, Procs: %d, Cores: %d",
		d.NumCount, d.NumTotal, d.NumPubs, d.NumSubs, runtime.NumGoroutine(), runtime.NumCPU())
	str += fmt.Sprintf("\n\t[License] Key: %s, Owner: %s", d.KeyLicense, d.Owner)
	str += fmt.Sprintf("\n\t[Time] Zone: %s, Start: %s, Now: %s, Expired: %v, Running: %v",
		d.TimeZone,
		d.AtCreated.Format("2006/01/02 15:04:05"), time.Now().Format("2006/01/02 15:04:05"),
		d.AtExpired.Format("2006/01/02 15:04:05"), time.Since(d.AtCreated))
	str += fmt.Sprintf("\n\t[Buffer] Pipe: %d (%d-%d), Slot: %d (%d-%d)",
		TRACK_MIN_BUFFERS, TRACK_MIN_BUFFERS, TRACK_MAX_BUFFERS,
		BUFFER_LEN_SLOTS, BUFFER_MIN_SLOTS, BUFFER_CAP_SLOTS)
	return
}

func (d *MothConfig) allowCORS(w http.ResponseWriter) {
	if d.CORSAllow {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
	}
}

func (d *MothConfig) allowCOP(w http.ResponseWriter) {
	if d.COPAllow {
		w.Header().Set("Cross-Origin-Embedder-Policy", "*") // unsafe-none | require-corp | credentialless
		w.Header().Set("Cross-Origin-Opener-Policy", "*")   // unsafe-none | same-origin-allow-popups | same-origin | noopener-allow-popups
	}
}

func (d *MothConfig) isValid() bool {
	return time.Now().Before(d.AtExpired)
}

// ---------------------------------------------------------------------------------
func (d *MothConfig) readConfig(object, fname string) (err error) {
	// log.Println("i.readConfig:", fname)

	file, err := os.Open(fname)
	if err != nil {
		// ignore to read config info from the file
		// log.Println("[Error]", err)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Println("[Error]", err)
		return
	}

	var config MothConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Println("[Error]", err, fname)
		return
	}

	if IsXidString(config.ID) {
		d.ID = config.ID // should be matched with its license key
	} else {
		d.ID = GetXidString()
	}
	if config.Name != "" {
		d.Name = config.Name
	}
	if config.TimeZone != "" {
		_, err = time.LoadLocation(config.TimeZone)
		if err != nil {
			log.Println(err)
		} else {
			d.TimeZone = config.TimeZone
		}
	}
	if config.KeyManager != "" {
		d.KeyManager = config.KeyManager
	}
	if config.KeyLicense != "" {
		d.KeyLicense = config.KeyLicense
	}

	if config.PortPlain > 0 {
		d.PortPlain = config.PortPlain
	}
	if config.PortSecure > 0 {
		d.PortSecure = config.PortSecure
	}
	if config.TCPPlain > 0 {
		d.TCPPlain = config.TCPPlain
	}
	if config.TCPSecure > 0 {
		d.TCPSecure = config.TCPSecure
	}
	if config.QUICSecure > 0 {
		d.QUICSecure = config.QUICSecure
	}
	if config.RaknetPlain > 0 {
		d.RaknetPlain = config.RaknetPlain
	}
	if config.KCPPlain > 0 {
		d.KCPPlain = config.KCPPlain
	}

	d.DirLog = config.DirLog
	if d.DirLog != "" {
		os.MkdirAll(d.DirLog, os.ModePerm)
	}
	d.DirData = config.DirData
	if d.DirData != "" {
		os.MkdirAll(d.DirData, os.ModePerm)
	}
	d.DirRecord = config.DirRecord
	if d.DirRecord != "" {
		os.MkdirAll(d.DirRecord, os.ModePerm)
	}
	d.SecHost = config.SecHost
	d.SecTicket = config.SecTicket
	d.CORSAllow = config.CORSAllow
	return
}

// ---------------------------------------------------------------------------------
func (d *MothConfig) writeConfig(object, fname string) (err error) {
	// log.Println("i.writeConfig:", fname)

	file, err := os.Create(fname)
	if err != nil {
		log.Println("[Error]", err)
		return
	}
	defer file.Close()

	data, _ := json.MarshalIndent(d, "", "   ")
	_, err = file.Write(data)
	if err != nil {
		log.Println("[Error]", err)
		return
	}
	return
}

//---------------------------------------------------------------------------------
// func (d *MothConfig) MarshalJSON() ([]byte, error) {
// 	type Alias MothConfig
// 	return json.Marshal(&struct {
// 		TypeRun int64 `json:",omitempty"`
// 		*Alias
// 	}{
// 		TypeRun: 0,
// 		Alias:   (*Alias)(d),
// 	})
// }

//=================================================================================
