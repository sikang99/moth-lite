// =================================================================================
// Filename: util-external.go
// Function: utility functions from external packages
// Copyright: TeamGRIT, 2022, 2025
// Author: Stoney Kang, sikang@teamgrit.kr
// =================================================================================
package main

import (
	"log"
	"net"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"gopkg.in/corvus-ch/zbase32.v1"
)

// ---------------------------------------------------------------------------------
func GetMachineID(appid string) (mid string) {
	var err error
	if appid == "" {
		mid, err = machineid.ID() // short serial number
	} else {
		mid, err = machineid.ProtectedID(appid) // quite long string
	}
	if err != nil {
		log.Println(err)
	}
	return
}

// ---------------------------------------------------------------------------------
func Zbase32EncodeByteToString(data []byte) (str string) {
	str = zbase32.StdEncoding.EncodeToString(data)
	return
}

// ---------------------------------------------------------------------------------
func Zbase32DecodeStringToByte(str string) (data []byte, err error) {
	data, err = zbase32.StdEncoding.DecodeString(str)
	if err != nil {
		log.Println(err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
func ListMACAddresses() {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Println(err)
		return
	}

	for _, iface := range interfaces {
		if iface.HardwareAddr == nil {
			continue
		}
		log.Println(iface.Name, iface.HardwareAddr.String())
	}
}

func CheckMACAddress(macStr string) (res bool) {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Println(err)
		return
	}

	for _, iface := range interfaces {
		if iface.HardwareAddr == nil {
			continue
		}
		if strings.ToLower(iface.HardwareAddr.String()) == strings.ToLower(macStr) {
			log.Println(iface.Name, iface.HardwareAddr.String(), "==", macStr)
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------------
func CheckMachineId(maddr string) (res bool) {
	mid := GetMachineID("")
	if mid == "" || maddr == "" {
		return false
	}
	return strings.Contains(mid, maddr)
}

//=================================================================================
