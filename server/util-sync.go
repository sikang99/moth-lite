// =================================================================================
// Filename: util-sync.go
// Function: Synchronization mechanisms
// Author: Stoney Kang, sikang@teamgrit.kr
// Copyright: TeamGRIT, 2021
// =================================================================================
package main

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// ---------------------------------------------------------------------------------
// [tidwall/spinlock](https://github.com/tidwall/spinlock) - A spinlock implementation for Go
// ---------------------------------------------------------------------------------
type SpinLock struct {
	_    sync.Mutex // for copy protection compiler warning
	lock uintptr
}

// If the lock is already in use, the calling goroutine blocks until the lock is available.
func (l *SpinLock) Lock() {
loop:
	if !atomic.CompareAndSwapUintptr(&l.lock, 0, 1) {
		runtime.Gosched()
		goto loop
	}
}

func (l *SpinLock) Unlock() {
	atomic.StoreUintptr(&l.lock, 0)
}

//=================================================================================
