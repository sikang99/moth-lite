package main

import (
	"fmt"
	"log"

	"github.com/denisbrodbeck/machineid"
)

func main() {
	mid := GetMachineID("teamgrit")
	fmt.Println("Machine ID:", mid)
}

func GetMachineID(appid string) (mid string) {
	var err error
	if appid == "" {
		mid, err = machineid.ID()
	} else {
		mid, err = machineid.ProtectedID(appid)
	}
	if err != nil {
		log.Println(err)
	}
	return
}
