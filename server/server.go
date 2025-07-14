// =================================================================================
// Filename: server.go
// Function: server management and control functions
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021-2024
// =================================================================================
package main

import (
	"net"
	"net/http"
	"sync/atomic"
)

// ---------------------------------------------------------------------------------
type ConnectionWatcher struct {
	nTotal int64
	nCount int64
}

func (cw *ConnectionWatcher) OnStateChange(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		atomic.AddInt64(&cw.nTotal, 1)
		atomic.AddInt64(&cw.nCount, 1)
	case http.StateHijacked, http.StateClosed:
		atomic.AddInt64(&cw.nCount, -1)
	}
}

func (cw *ConnectionWatcher) Count() int {
	return int(atomic.LoadInt64(&cw.nCount))
}

//=================================================================================
