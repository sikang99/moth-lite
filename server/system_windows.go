//go:build windows
// +build windows

// =================================================================================
// Filename: system_windows.go
// Function: system information handling
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2024
// =================================================================================
package main

import (
	"fmt"
	"log"
	"syscall"
	"unsafe"
)

// ---------------------------------------------------------------------------------
func SetSysRlimit() (err error) {
	log.Println("i.SetSysRlimit", "windows")
	// TBD
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
	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	_, _, err = syscall.NewLazyDLL("kernel32.dll").NewProc("GetDiskFreeSpaceExW").Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	if err != nil {
		return nil, err
	}

	disk = &DiskStatus{
		Type:  "disk",
		All:   totalNumberOfBytes,
		Avail: freeBytesAvailable,
		Free:  totalNumberOfFreeBytes,
	}
	disk.Used = disk.All - disk.Free
	return disk, nil
}

//=================================================================================
