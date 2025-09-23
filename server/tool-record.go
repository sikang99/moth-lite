// =================================================================================
// Filename: tool-record.go
// Function: media (audio, video) recording functions
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2022
// =================================================================================
package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------------
func (d *Channel) startMediaRecord(source, vtrack, atrack string) (err error) {
	log.Println("i.startMediaRecord:", d.ID)

	d.RecordState = Using
	defer func() {
		d.RecordState = Idle
		d.recordCmd = nil
	}()

	pStudio.pushEvent("record-in", d.Name, d.ID)
	defer pStudio.pushEvent("record-out", d.Name, d.ID)

	// -- NOTICE: wait time for ready media
	time.Sleep(time.Second)

	pgname := "tools/moth-med-record"
	pgopts := fmt.Sprintf(" -channel=%s -source=%s", d.ID, source)
	pgopts += fmt.Sprintf(" -vtrack=%s -atrack=%s", vtrack, atrack)
	pgopts += fmt.Sprintf(" -file=%s/moth-%s-%s", mConfig.DirRecord, d.ID, time.Now().Format("20060102150405"))
	log.Println("[RECORD]", pgname, pgopts)

	params := strings.Fields(pgopts)
	d.recordCmd = exec.Command(pgname, params...)
	out, err := d.recordCmd.CombinedOutput()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(out))
	return
}

// ---------------------------------------------------------------------------------
func (d *Channel) stopMediaRecord(source, vtrack, atrack string) (err error) {
	log.Println("i.stopMediaRecord:", d.ID)

	if d.recordCmd != nil {
		err = d.recordCmd.Process.Kill()
	}
	return
}

// ---------------------------------------------------------------------------------
func (d *Channel) startDataRecord(source, track string) (err error) {
	log.Println("i.startDataRecord:", d.ID)

	if d.State != Using {
		err = fmt.Errorf("channel %snot in use", d.ID)
		return
	}

	d.RecordState = Using
	defer func() {
		d.RecordState = Idle
	}()

	pStudio.pushEvent("record-in", d.Name, d.ID)
	defer pStudio.pushEvent("record-out", d.Name, d.ID)

	for d.State == Using && d.RecordState == Using {
		time.Sleep(time.Second)
	}

	return
}

// ---------------------------------------------------------------------------------
func (d *Channel) stopDataRecord(source, track string) (err error) {
	log.Println("i.stopDataRecord:", d.ID)

	d.RecordState = Idle
	return
}

//=================================================================================
