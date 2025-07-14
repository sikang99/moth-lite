// =================================================================================
// Filename: api-pang-http.go
// Function: pang http API for message based streaming
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021
// =================================================================================
package main

import (
	"log"
	"net/http"
)

// ---------------------------------------------------------------------------------
func PangLivePublisher(w http.ResponseWriter, r *http.Request, qo QueryOption) (err error) {
	log.Println("IN PangLivePublisher:", r.URL)
	defer log.Println("OUT PangLivePublisher:", r.URL, err)

	// TBD
	return
}

// ---------------------------------------------------------------------------------
func PangLiveSubscriber(w http.ResponseWriter, r *http.Request, qo QueryOption) (err error) {
	log.Println("IN PangLiveSubscriber:", r.URL)
	defer log.Println("OUT PangLiveSubscriber:", r.URL, err)
	// TBD
	return
}

//=================================================================================
