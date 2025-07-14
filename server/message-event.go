// =================================================================================
// Filename: message-event.go
// Function: Signalling server for moth
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021
// =================================================================================
package main

import (
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------------
type EventMessage struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Data      string    `json:"data,omitempty"`
	Path      string    `json:"path,omitempty"`
	RequestID string    `json:"req_id,omitempty"`
	AtCreated time.Time `json:"at_created"`
}

func (d EventMessage) String() (str string) {
	str = fmt.Sprintf("[%s] %s, %s, %s, %s, %s", d.Type, d.ID, d.Name, d.Data, d.Path, d.RequestID)
	// str += fmt.Sprintf("\tData: %s, %s", d.Data, d.AtCreated.Format("2006/01/02 15:04:05"))
	return
}

func NewEventMessagePointer() (d *EventMessage) {
	d = &EventMessage{Type: "event", AtCreated: time.Now()}
	return
}

//=================================================================================
