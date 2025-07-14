// =================================================================================
// Filename: query-option.go
// Function: processing options from clients
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-23, 2025
// =================================================================================
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// ---------------------------------------------------------------------------------
type Addr struct {
	Network string `json:"network,omitempty"` // tcp/udp, tcp4/udp4, tcp6/udp6
	Host    string `json:"host,omitempty"`
	Port    string `json:"port,omitempty"`
	Mime    string `json:"mime,omitempty"`
}

type QueryOption struct {
	URL struct {
		Scheme string
		Path   string
		Addr   string
	}
	// Addr    Addr   `json:"addr,omitempty"`
	Stream struct {
		Addr Addr   `json:"addr,omitempty"`
		Mime string `json:"mime,omitempty"`
		Role string `json:"role,omitempty"` // role: pub, sub, meb
	}
	Format  string `json:"format,omitempty"`
	Session struct {
		ID      string `json:"id,omitempty"`
		Name    string `json:"name,omitempty"`
		Unit    string `json:"unit,omitempty"`
		Timeout int    `json:"timeout,omitempty"`
		Wait    string `json:"wait,omitempty"`  // allow wait a track on no pubs
		Style   string `json:"style,omitempty"` // allow multiple pubs for the same track
		ReqID   string `json:"req_id,omitempty"`
	} `json:"session,omitempty"`
	Channel struct {
		ID     string `json:"id,omitempty"`
		Name   string `json:"name,omitempty"`
		Style  string `json:"style,omitempty"`
		Key    string `json:"key,omitempty"`
		Code   string `json:"code,omitempty"`
		Record string `json:"record,omitempty"`
		Trans  string `json:"trans,omitempty"`
		Period string `json:"period,omitempty"`
	} `json:"channel,omitempty"`
	Source struct {
		Label string `json:"label,omitempty"`
	} `json:"source,omitempty"`
	Track struct {
		Label    string `json:"label,omitempty"`
		Mode     string `json:"mode,omitempty"`     // operation mode: single, (bundle), parallel, secure
		Style    string `json:"style,omitempty"`    // style: (mono), multi
		Filter   string `json:"filter,omitempty"`   // filter: echo, each, (group), all = echo + group
		Codec    string `json:"codec,omitempty"`    // requiring codec name
		Proc     string `json:"proc,omitempty"`     // function to process
		Bitrate  string `json:"bitrate,omitempty"`  // need?
		Parallel int    `json:"parallel,omitempty"` // number of parallel streams within a track
	} `json:"track,omitempty"`
	Buffer struct {
		Label string `json:"label,omitempty"`
		Total int    `json:"total,omitempty"` // total number of buffers
		Order int    `json:"order,omitempty"` // order number in buffers
		Cap   int    `json:"cap,omitempty"`   // number of slots to allocate
		Len   int    `json:"len,omitempty"`   // number of slots to use
	} `json:"buffer,omitempty"`
}

func (d QueryOption) String() (str string) {
	str += fmt.Sprintf("QueryOption:")
	str += fmt.Sprintf("\n\t[URL] Scheme: %s, Path: %s, Addr: %s",
		d.URL.Scheme, d.URL.Path, d.URL.Addr)
	str += fmt.Sprintf("\n\t[Channel: %s,%s], [Source: %s], [Track: %s,%s]",
		d.Channel.ID, d.Channel.Name, d.Source.Label, d.Track.Label, d.Track.Mode)
	str += fmt.Sprintf("\n\t[Buffer] Total: %d, Len: %d, [Session] Clock: %s Timeout: %d",
		d.Buffer.Total, d.Buffer.Len, d.Session.Unit, d.Session.Timeout)
	return
}

// ---------------------------------------------------------------------------------
func GetQueryOptionFromString(scheme, qpath, qstr string) (qo QueryOption, err error) {
	// log.Println("i.GetQueryOptionFromString:", qpath, qstr)

	query, err := url.ParseQuery(qstr)
	if err != nil {
		log.Println(err)
		return
	}

	qo, err = GetQueryOptionFromQuery(query)
	if err != nil {
		log.Println(err)
		return
	}

	qo.URL.Scheme = scheme
	qo.URL.Path = qpath
	return
}

// ---------------------------------------------------------------------------------
func GetQueryOptionFromRequest(r *http.Request) (qo QueryOption, err error) {
	// log.Println("i.GetQueryOptionFromRequest:", r.URL.Path, r.URL.RawQuery)

	query := r.URL.Query()
	qo, err = GetQueryOptionFromQuery(query)
	if err != nil {
		log.Println(err)
		return
	}

	qo.URL.Scheme = r.URL.Scheme
	qo.URL.Path = r.URL.Path
	qo.URL.Addr = r.RemoteAddr
	return
}

// ---------------------------------------------------------------------------------
func GetQueryOptionFromQuery(query url.Values) (qo QueryOption, err error) {
	// log.Println("i.GetQueryOptionFromQuery:", query)

	qo.Stream.Addr.Host = query.Get("host")
	qo.Stream.Addr.Port = query.Get("port")
	qo.Stream.Role = query.Get("role")

	qo.Format = query.Get("format")
	if qo.Format == "" {
		qo.Format = "text"
	}

	qo.Channel.ID = query.Get("channel")    // ID or Style (instant)
	qo.Channel.Name = query.Get("name")     // Name
	qo.Channel.Key = query.Get("key")       // StreamKey
	qo.Channel.Record = query.Get("record") // Recording (on|off)
	qo.Channel.Trans = query.Get("trans")   // Transcoding (on|off)
	qo.Channel.Period = query.Get("period") // Time period in hour string

	// channel style: (static), instant, dynamic, cf) should be removed after testing)
	// ch := pStudio.setChannelByQueryOption(qo)
	// if ch != nil {
	// 	qo.Channel.ID = ch.ID
	// }

	// Channel, Source, Track control
	qo.Source.Label = query.Get("source") // Source name (label)
	if qo.Source.Label == "" {
		qo.Source.Label = "base" // (base), addon, ...
	}
	qo.Track.Label = query.Get("track") // Track name (label) in a Source
	if qo.Track.Label == "" {
		qo.Track.Label = "video" // (video), audio, data, ...
	}
	qo.Track.Mode = query.Get("mode") // mode type of a Track
	if qo.Track.Mode == "" {
		qo.Track.Mode = "single" // (single), bundle, 2022/06/17: go back, shoot (wt/datagram)
	}
	qo.Track.Style = query.Get("style") // mode type of a Track
	if qo.Track.Style == "" {
		qo.Track.Style = "mono" // (mono), multi
	}
	qo.Track.Filter = query.Get("filter") // mode type of a Track
	if qo.Track.Filter == "" {
		qo.Track.Filter = "group" // echo, self, (group), all
	}
	qo.Track.Codec = query.Get("codec") // jpeg, vp8, h264, aac, ...
	qo.Track.Proc = query.Get("proc")   // motion, face, yolo, aes, ...
	qo.Track.Bitrate = query.Get("bitrate")
	// log.Println(qo.Track.Bitrate)
	// qo.Track.Mime = query.Get("mime") // mime type of a Track content
	// log.Println(qo.Track.Mime)
	parallel := query.Get("parallel")
	if parallel != "" {
		n, err := strconv.Atoi(parallel)
		if err != nil {
			log.Println(err)
		} else {
			qo.Track.Parallel = n
			log.Println("parallel:", qo.Track.Parallel)
		}
	}

	qo.Session.ReqID = query.Get("reqid") // unique info provided by client
	qo.Session.ID = query.Get("session")
	qo.Session.Wait = query.Get("wait")

	qo.Session.Unit = query.Get("unit") // time unit for buffering check
	if qo.Session.Unit == "" {
		qo.Session.Unit = "milli" // (milli), micro, nano, zero
	}

	timeout := query.Get("timeout") // waiting time to receive
	if timeout != "" {
		n, err := strconv.Atoi(timeout)
		if err != nil {
			log.Println(err)
		} else {
			qo.Session.Timeout = n
		}
	}

	// --- Buffer dynamic control
	total := query.Get("buf_total") // Total number of buffers to be used
	if total != "" {
		n, err := strconv.Atoi(total)
		if err != nil {
			log.Println(err)
		} else {
			qo.Buffer.Total = n
			log.Println("buf_total:", qo.Buffer.Total)
		}
		order := query.Get("buf_order") // Buffer order number
		if order != "" {
			n, err := strconv.Atoi(order)
			if err != nil {
				log.Println(err)
			} else {
				qo.Buffer.Order = n
				log.Println("buf_order:", qo.Buffer.Order)
			}
		}
	}
	cap := query.Get("buf_cap") // track buffer capa = N of buffer slots to allocate
	if cap != "" {
		n, err := strconv.Atoi(cap)
		if err != nil {
			log.Println(err)
		} else {
			qo.Buffer.Cap = n
			log.Println("buf_cap:", qo.Buffer.Cap)
		}
	}
	len := query.Get("buf_len") // track buffer length = N of buffer slots to use actually
	if len != "" {
		n, err := strconv.Atoi(len)
		if err != nil {
			log.Println(err)
		} else {
			qo.Buffer.Len = n
			log.Println("buf_len:", qo.Buffer.Len)
		}
	}

	// prepare a channel with given query options
	ch := pStudio.setChannelByQueryOption(qo)
	if ch != nil {
		qo.Channel.ID = ch.ID
	}
	return
}

//=================================================================================
