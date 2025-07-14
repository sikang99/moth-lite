//go:build linux || darwin
// +build linux darwin

// =================================================================================
// Filename: system.go
// Function: system information handling
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2022 - 2024
// =================================================================================
package main

import (
	"fmt"
	"log"
	"syscall"
)

// ---------------------------------------------------------------------------------
func SetSysRlimit() (err error) {
	log.Println("i.SetSysRlimit")
	var rLimit syscall.Rlimit
	if err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Println(err)
		return
	}
	rLimit.Cur = rLimit.Max
	if err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Println(err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
type DiskStatus struct {
	Type  string `json:"type"`
	All   uint64 `json:"all"`
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
	Avail uint64 `json:"avail"`
}

func (d *DiskStatus) String() (str string) {
	str = fmt.Sprintf("[%s] All: %.2f GB, ", d.Type, float64(d.All)/float64(GB))
	str += fmt.Sprintf("Avail: %.2f GB, ", float64(d.Avail)/float64(GB))
	str += fmt.Sprintf("Used: %.2f GB, ", float64(d.Used)/float64(GB))
	str += fmt.Sprintf("Free: %.2f GB, ", float64(d.Free)/float64(GB))
	str += fmt.Sprintf("Spare ratio: %.2f percent", (float64(d.Avail)/float64(d.All))*100)
	return
}

func GetDiskUsage(path string) (disk *DiskStatus, err error) {
	fs := syscall.Statfs_t{}
	err = syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk = &DiskStatus{
		Type:  "disk",
		All:   fs.Blocks * uint64(fs.Bsize),
		Avail: fs.Bavail * uint64(fs.Bsize),
		Free:  fs.Bfree * uint64(fs.Bsize),
	}
	disk.Used = disk.All - disk.Free
	return
}

//=================================================================================
