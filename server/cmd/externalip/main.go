package main

/*
URL: http://myexternalip.com/#golang
*/

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

//---------------------------------------------------------------------------------
func main() {
	resp, err := http.Get("https://ipecho.net/plain")
	if err != nil {
		log.Println(err)
		resp, err = http.Get("http://myexternalip.com/raw")
		if err != nil {
			log.Println(err)
			return
		}
	}
	defer resp.Body.Close()

	ipaddr, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	exip := net.ParseIP(string(ipaddr))
	if IsPublicIP(exip) {
		fmt.Println("External IP:", exip)
	}
}

//---------------------------------------------------------------------------------
func IsPublicIP(IP net.IP) bool {
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	if ip4 := IP.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	return false
}

//=================================================================================
