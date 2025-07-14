// =================================================================================
// Filename: manager-http.go
// Function: manager http API handling
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2020-2021
// =================================================================================
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// ---------------------------------------------------------------------------------
func SendHTTPResponse(w http.ResponseWriter, format, str string, err error) {
	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if format == "json" {
			fmt.Fprintf(w, "{ \"error\": \"%s\" }", err)
		} else {
			fmt.Fprintf(w, "Error> %s", err)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		if str == "" {
			str = "Ok!"
			if format == "json" {
				str = fmt.Sprintf("{ \"result\" : \"%s\" }", str)
			}
		}
		fmt.Fprintf(w, "%s", str)
	}
}

// ---------------------------------------------------------------------------------
func ProcManagerHttpCommand(w http.ResponseWriter, r *http.Request) (str string, err error) {
	log.Println("i.ProcManagerHttpCommand:")

	s := pStudio.addNewSessionWithName(r.URL.Path)
	defer pStudio.deleteSession(s)

	cmd := NewCommandPointer()
	defer func() {
		SendHTTPResponse(w, cmd.Format, str, err)
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		return
	}
	cmd.Data = string(body)

	err = cmd.parseQuery(r)
	if err != nil {
		log.Println("cmd.parseQuery:", err)
		return
	}
	log.Println(cmd)

	err = cmd.checkManagerPermission()
	if err != nil {
		log.Println("cmd.checkManagerPermission:", err)
		return
	}

	str, err = cmd.execManager()
	if err != nil {
		log.Println("cmd.execManager:", err)
		return
	}
	return
}

// ---------------------------------------------------------------------------------
func ProcManagerHttpCommand2(w http.ResponseWriter, r *http.Request) (str string, err error) {
	log.Println("i.ProcManagerHttpCommand2:")

	s := pStudio.addNewSessionWithName(r.URL.Path)
	defer pStudio.deleteSession(s)

	cmd := NewCommandPointer()
	defer func() {
		SendHTTPResponse(w, cmd.Format, str, err)
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("readAll: %s", err)
		return
	}

	err = json.Unmarshal(body, cmd)
	if err != nil {
		err = fmt.Errorf("unmarshal: %s", err)
		return
	}

	str, err = cmd.execManager2()
	if err != nil {
		err = fmt.Errorf("execManager2: %s", err)
		return
	}
	return
}

//=================================================================================
