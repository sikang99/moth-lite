// =================================================================================
// Filename: p2p-hole.go
// Function: Hole punching for p2p communication
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021
// =================================================================================
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

const (
	PUNCH_SERVER_PORT = 9999
)

// ---------------------------------------------------------------------------------
var userIP map[string]string

type ChatRequest struct {
	Action   string
	Username string
	Message  string
}

// ---------------------------------------------------------------------------------
func HolePunchServerForTCP() {

}

// ---------------------------------------------------------------------------------
func HolePunchServerForUDP() {
	log.Println("IN HolePunchServerForUDP:")
	defer log.Println("OUT HolePunchServerForUDP:")

	userIP = map[string]string{}
	servAddr := fmt.Sprintf(":%d", PUNCH_SERVER_PORT)
	udpAddr, err := net.ResolveUDPAddr("udp4", servAddr)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatal(err)
	}

	for {
		handleClient(conn)
	}
}

/*
Action:

	New -- Add a new user
	Get -- Get a user IP address

Username:

	New -- New user's name
	Get -- The requested user name
*/
func handleClient(conn *net.UDPConn) {
	var buf [2048]byte

	n, addr, err := conn.ReadFromUDP(buf[0:])
	if err != nil {
		return
	}

	var chatRequest ChatRequest
	err = json.Unmarshal(buf[:n], &chatRequest)
	if err != nil {
		log.Print(err)
		return
	}

	switch chatRequest.Action {
	case "New":
		remoteAddr := fmt.Sprintf("%s:%d", addr.IP, addr.Port)
		fmt.Println(remoteAddr, "connecting")
		userIP[chatRequest.Username] = remoteAddr

		// Send message back
		messageRequest := ChatRequest{
			"Chat",
			chatRequest.Username,
			remoteAddr,
		}
		jsonRequest, err := json.Marshal(&messageRequest)
		if err != nil {
			log.Print(err)
			break
		}
		conn.WriteToUDP(jsonRequest, addr)
	case "Get":
		// Send message back
		peerAddr := ""
		if _, ok := userIP[chatRequest.Message]; ok {
			peerAddr = userIP[chatRequest.Message]
		}

		messageRequest := ChatRequest{
			"Chat",
			chatRequest.Username,
			peerAddr,
		}
		jsonRequest, err := json.Marshal(&messageRequest)
		if err != nil {
			log.Print(err)
			break
		}
		_, err = conn.WriteToUDP(jsonRequest, addr)
		if err != nil {
			log.Print(err)
		}
	}
	fmt.Println("User table:", userIP)
}

// ---------------------------------------------------------------------------------
func HolePunchClientForUDP(cliAddr, servAddr string, username, peername string) {
	log.Println("IN HolePunchClientForUDP:", cliAddr, servAddr, username, peername)
	defer log.Println("OUT HolePunchClientForUDP:")

	buf := make([]byte, 2048)

	// Prepare the register user to server.
	saddr, err := net.ResolveUDPAddr("udp4", servAddr)
	if err != nil {
		log.Print("Resolve address failed for server:", servAddr)
		log.Fatal(err)
	}

	// Prepare local multcast address.
	caddr, err := net.ResolveUDPAddr("udp4", cliAddr)
	if err != nil {
		log.Print("Resolve local address failed for client:", cliAddr)
		log.Fatal(err)
	}
	conn, err := net.ListenUDP("udp", caddr)
	if err != nil {
		log.Print("Listen failed for UDP:", caddr)
		log.Fatal(err)
	}

	// Send registration information to server.
	initChatRequest := ChatRequest{
		"New",
		username,
		"",
	}
	jsonRequest, err := json.Marshal(initChatRequest)
	if err != nil {
		log.Print("Marshal Register information failed.")
		log.Fatal(err)
	}
	_, err = conn.WriteToUDP(jsonRequest, saddr)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Waiting for server response...")
	_, _, err = conn.ReadFromUDP(buf)
	if err != nil {
		log.Print("Register to server failed.")
		log.Fatal(err)
	}

	// Send connect request to server
	connectChatRequest := ChatRequest{
		"Get",
		username,
		peername,
	}
	jsonRequest, err = json.Marshal(connectChatRequest)
	if err != nil {
		log.Print("Marshal connection information failed.")
		log.Fatal(err)
	}

	var serverResponse ChatRequest
	for i := 0; i < 3; i++ {
		conn.WriteToUDP(jsonRequest, saddr)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Print("Get peer address from server failed.")
			log.Fatal(err)
		}
		err = json.Unmarshal(buf[:n], &serverResponse)
		if err != nil {
			log.Print("Unmarshal server response failed.")
			log.Fatal(err)
		}
		if serverResponse.Message != "" {
			break
		}
		time.Sleep(10 * time.Second)
	}
	if serverResponse.Message == "" {
		log.Fatal("Cannot get peer's address")
	}
	log.Print("Peer address: ", serverResponse.Message)
	peerAddr, err := net.ResolveUDPAddr("udp4", serverResponse.Message)
	if err != nil {
		log.Print("Resolve peer address failed.")
		log.Fatal(err)
	}

	// Start chatting.
	go listen(conn)
	for {
		fmt.Print("Input message: ")
		message := make([]byte, 2048)
		fmt.Scanln(&message)
		messageRequest := ChatRequest{
			"Chat",
			username,
			string(message),
		}
		jsonRequest, err = json.Marshal(messageRequest)
		if err != nil {
			log.Print("Error: ", err)
			continue
		}
		conn.WriteToUDP(jsonRequest, peerAddr)
	}
}

func listen(conn *net.UDPConn) {
	for {
		buf := make([]byte, 2048)
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Print(err)
			continue
		}
		// log.Print("Message from ", addr.IP)

		var message ChatRequest
		err = json.Unmarshal(buf[:n], &message)
		if err != nil {
			log.Print(err)
			continue
		}
		fmt.Println(message.Username, ":", message.Message)
	}
}

//=================================================================================
