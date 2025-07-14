// =================================================================================
// Filename: util-id.go
// Function: identity system of spider, moth
// Copyright: TeamGRIT, 2020
// Author: Stoney Kang, sikang@teamgrit.kr
// =================================================================================
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/rs/xid"
)

// ---------------------------------------------------------------------------------
func GetXid() xid.ID {
	id := xid.New()
	return id
}

func GetXidString() string {
	id := xid.New()
	return id.String()
}

func IsXidString(xstr string) bool {
	_, err := xid.FromString(xstr)
	return err == nil
}

func GetXidStringWithTime(t time.Time) string {
	id := xid.NewWithTime(t)
	return id.String()
}

func GetXidStringTime(xstr string) time.Time {
	id, _ := xid.FromString(xstr)
	return id.Time()
}

func GetXidStringInfo(xstr string) (str string) {
	id, err := xid.FromString(xstr)
	if err != nil {
		log.Println(err)
		return
	}
	return fmt.Sprintf("[%s] Machine: %x, Pid: %d, Time: %s, Counter: %d",
		xstr, id.Machine(), id.Pid(), id.Time().Format("2006/01/02 15:04:05"), id.Counter())
}

func PrintXidStringInfo(xstr string) {
	fmt.Println(GetXidStringInfo(xstr))
}

//---------------------------------------------------------------------------------
