// =================================================================================
// Filename: manager-cmd2.go
// Function: command processing for manager api
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2020-2021
// =================================================================================
package main

import "fmt"

// --------------------------------------------------------------------------------
func (cmd *Command) execManager2() (str string, err error) {
	// log.Println("execManager2:", cmd.Op)

	switch cmd.Op {
	default:
		err = fmt.Errorf("unknown command op: %s", cmd.Op)
	}
	return
}

//=================================================================================
