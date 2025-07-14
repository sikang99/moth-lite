// =================================================================================
// Filename: util-license.go
// Function: license key handling
// Copyright: TeamGRIT, 2022-2025
// Author: Stoney Kang, sikang@teamgrit.kr
// =================================================================================
package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	STR_BASE_TIME  = "2006-01-02"
	STR_START_TIME = "2023-01-01"
)

// ---------------------------------------------------------------------------------
type License struct {
	xid  string // unique server id
	addr string // ip or mac address
	note string // note (D or S) for extension
	ver  int    // version types (1, 2, 3 at 2025-01-26)
	code int    // site code = country code + serial number
	site string // site name
	seq  int    // sequence number
	chk  int    // checking method : nothing(0), external ip(1), local ip(2), mac address(3), machine id(4)
	pubs int    // N of publishers (casters)
	subs int    // N of subscribers (players)
	days int    // validation days
	date string // validation date
}

func (d *License) String() (str string) {
	str += fmt.Sprintf("[license] xid:%s, addr:%s, note:%s, ver:%d, chk:%d, code:%d, seq:%d, pubs:%d, subs:%d, days:%d",
		d.xid, d.addr, d.note, d.ver, d.chk, d.code, d.seq, d.pubs, d.subs, d.days)
	str += fmt.Sprintf("\n\tsite:%s, exp date:%s", d.site, d.date)
	return
}

// ---------------------------------------------------------------------------------
func (d *License) genKeyStringByInput() (lkstr string) {
	log.Println("i.genKeyStringByInput:")

	d.xid = GetXidString()
	// d.addr = GetExternalIPString()
	// 3.36.60.95(dev) 3.35.83.222(prod) 211.219.34.241 (office) 182.212.177.58 (home), 49.50.162.98 (cobot.center)
	// d.addr = "192.168.0.2" // "3.35.214.102", 49.50.162.98" // no fixed ip
	d.addr = "48:21:0b:6f:ab:87"
	//d.addr = "C40F767AE65B"
	// d.addr = "211.195.53.220"

	d.note = "D"  // D(umb) or S(pecial), only D is used
	d.ver = 3     // current license version: v3 (2025-01-01, v1.1.6.8)
	d.seq = 1     // sequence number of the install site
	d.chk = 3     // checking method : nothing(0), external ip(1), local ip(2), mac address(3), machine id(4)
	d.code = 8207 // 8291,8276->8277 : korea/teamgrit
	d.pubs = 10   // 1 - 999
	d.subs = 1000 // 1 - 9999

	switch d.ver {
	case 1:
		d.days = 3650 // 3650 = 10 years
	case 2, 3:
		d.date = "2036-04-01"
		d.days = d.calcDaysFromBaseByDate(d.date)
	}

	lkstr = d.makeKeyString()
	return
}

// ---------------------------------------------------------------------------------
func (d *License) calcDaysFromBaseByDate(date string) (days int) {
	// log.Println("i.calcDaysFromBaseByDate:", date)

	stime, _ := time.Parse(STR_BASE_TIME, STR_START_TIME)
	etime, err := time.Parse(STR_BASE_TIME, date)
	if err != nil {
		log.Println(err)
		return
	}
	days = int(etime.Sub(stime).Hours() / 24)
	log.Println("days:", days, "until", date)
	return
}

// ---------------------------------------------------------------------------------
func (d *License) calcDateFromDays(days int) (date string) {
	// log.Println("i.calcDateFromDays:", days)

	stime, _ := time.Parse(STR_BASE_TIME, STR_START_TIME)
	etime := stime.AddDate(0, 0, days)
	date = etime.Format(STR_BASE_TIME)
	return
}

// ---------------------------------------------------------------------------------
func (d *License) makeKeyString() (lkstr string) {
	log.Println("i.makeKeyString:", d.ver, d.chk, d.days, d.code)

	exip := GetExternalIPString()
	loip := GetLocalIPString()
	log.Println("Address:", d.addr, "exip:", exip, "loip:", loip)

	xdstr := fmt.Sprintf("%s-%s-%1s%1d%04d%02d%02d%04d%04d%04d",
		d.xid, d.addr, d.note, d.ver, d.code, d.seq, d.chk, d.pubs, d.subs, d.days)
	// log.Println(xdstr)

	hash := md5.Sum([]byte(xdstr))
	hstr := hex.EncodeToString(hash[:])

	if d.ver < 3 {
		lkstr = fmt.Sprintf("%s:%s", xdstr, hstr[:8])
	} else {
		lkstr = fmt.Sprintf("%s!%s", xdstr, hstr[:8])
	}
	// log.Println(lkstr)
	return
}

// ---------------------------------------------------------------------------------
func (d *License) readKeyString(lkstr string) (err error) {
	// log.Println("i.readKeyString:")

	// license key format: v2 [xid]-[addr]-[nums]:[hash]
	// license key format: v3 [xid]-[addr]-[nums]![hash]
	parts := strings.Split(lkstr, "!") // v3
	if len(parts) < 2 {
		err = fmt.Errorf("invalid license format: %s", lkstr)
		return
	}

	hash := md5.Sum([]byte(parts[0]))
	hstr := hex.EncodeToString(hash[:])

	if hstr[0:8] != parts[1] {
		err = fmt.Errorf("invalid license key hash: %s", parts[1])
		return
	}

	// key information format: [xid]-[addr]-[nums]
	toks := strings.Split(parts[0], "-")
	if len(toks) < 3 {
		err = fmt.Errorf("invalid license key format: %s", parts[0])
		return
	}

	d.xid = toks[0]
	if !IsXidString(d.xid) {
		err = fmt.Errorf("invalid xid: %s", d.xid)
		return
	}

	d.addr = toks[1]

	// license numbers format: [nums]
	_, err = fmt.Sscanf(toks[2], "%1s%1d%04d%02d%02d%04d%04d%04d",
		&d.note, &d.ver, &d.code, &d.seq, &d.chk, &d.pubs, &d.subs, &d.days)
	if err != nil {
		log.Println(err)
		return
	}

	d.date = d.calcDateFromDays(d.days)
	d.site = d.codeString()
	// log.Println(d)
	return
}

// ---------------------------------------------------------------------------------
func (d *License) parseKeyString(lkstr string) (err error) {
	log.Println("i.parseKeyString:", lkstr)

	err = d.readKeyString(lkstr)
	if err != nil {
		log.Println(err)
		return
	}

	// part 2: checking method
	switch d.chk {
	case 0: // nothing to check
		// -- ignore the checking
	case 1: // external ip
		if d.addr != GetExternalIPString() {
			err = fmt.Errorf("invalid external ip: %s", d.addr)
			return
		}
	case 2: // local ip
		if d.addr != GetLocalIPString() {
			err = fmt.Errorf("invalid local ip: %s", d.addr)
			return
		}
	case 3: // mac address
		if !CheckMACAddress(d.addr) {
			err = fmt.Errorf("invalid mac address: %s", d.addr)
			return
		}
	case 4: // machine id
		if !CheckMachineId(d.addr) {
			err = fmt.Errorf("invalid machine id: %s", d.addr)
			return
		}
	default:
		err = fmt.Errorf("invalid check method: %d", d.chk)
		return
	}

	// part 3: license internal numbers
	if d.note != "D" { // D(umb) or S(pecial)
		err = fmt.Errorf("invalid note: %s", d.note)
		return
	}

	if !d.isValidNumbers() {
		err = fmt.Errorf("invalid code: %d, %d", d.ver, d.code)
		return
	}

	switch d.ver {
	case 1:
		mConfig.AtExpired = mConfig.AtCreated.AddDate(0, 0, d.days)
	case 2, 3:
		stime, err := time.Parse(STR_BASE_TIME, STR_START_TIME)
		if err != nil {
			log.Println(err)
			return err
		}
		mConfig.AtExpired = stime.AddDate(0, 0, d.days)
	default:
		err = fmt.Errorf("invalid license version: %d", d.ver)
		return
	}
	// log.Println(d.ver, d.days, mConfig.AtExpired)

	// part 1: checking for license xid and server xid
	if mConfig.ID != d.xid {
		err = fmt.Errorf("not matched server xid: %s", d.xid)
		return
	}

	mConfig.NumPubs = d.pubs
	mConfig.NumSubs = d.subs
	mConfig.Owner = d.site
	return
}

// ---------------------------------------------------------------------------------
// func (d *License) checkKeyInformation() (err error) {
// 	// TBD
// 	return
// }

// ---------------------------------------------------------------------------------
// func (d *License) setKeyInformationToConfig() (err error) {
// 	// TBD
// 	return
// }

// ---------------------------------------------------------------------------------
func (d *License) isValidNumbers() (ret bool) {
	if d.ver < 1 || d.ver > 3 {
		log.Println("invalid version:", d.ver)
		return
	}
	if d.codeString() == "unknown" {
		log.Println("invalid site code:", d.code)
		return
	}
	if d.pubs > 1000 || d.subs > 10000 { // overload numbers on a single machine
		log.Println("invalid number of pubs or subs:", d.pubs, d.subs)
		return
	}
	if d.days > 9000 { // WARNING: 3650 (10 years) + gap days from 2023-01-01 until now
		log.Println("invalid validation days:", d.days)
		return
	}
	return true
}

// ---------------------------------------------------------------------------------
func (d *License) codeString() (str string) {
	switch d.code {
	// --- Japan (81)
	case 8101:
		str = "japan/kcme"
	case 8102:
		str = "japan/lc2lab"
	case 8103: // https://remoterobotics.co.jp/
		str = "japan/r2"
	case 8104: // https://next-comming.jp/
		str = "japan/coming"
	// --- Korea (82)
	case 8201: // http://eansoft.co.kr/
		str = "korea/eansoft"
	case 8202: // http://goodganglabs.com/
		str = "korea/goodganglabs"
	case 8203: // https://www.jbnu.ac.kr/kor/
		str = "korea/jbnu"
	case 8204: // https://vestek.co.kr/
		str = "korea/vestek"
	case 8205: // https://http://www.kanavi-mobility.com/
		str = "korea/kanavi"
	case 8206: // https://www.krri.re.kr (railway)
		str = "korea/krri"
	case 8207: // https://lg-pri.com/
		str = "korea/lgpri"
	case 8208: // https://fire.ulsan.go.kr/, 2025/02/06
		str = "korea/fire-ulsan"
	case 8209: // https://nics.me.go.kr/, 2025/03/31
		str = "korea/nics"
	case 8210: // https://roas.co.kr/, 2025/06/18
		str = "korea/roas"
	case 8211, 8212, 8213, 8214: // https://next-coming.kr/
		str = "korea/coming"
	case 8291, 8276: // http://teamgrit/kr, 8266 for Spider
		str = "korea/teamgrit"
	case 8292, 8277: // http://teamgrit/kr, 8267 for Spider
		str = "korea/teamgrit-lab"
	default:
		str = "unknown"
	}
	return
}

//==================================================================================
