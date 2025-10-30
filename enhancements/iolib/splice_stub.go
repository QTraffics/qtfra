//go:build !linux

package iolib

import "syscall"

func copySplice(source syscall.RawConn, destination syscall.RawConn,
	readCounters []counter.Func, writeCounters []counter.Func,
) (handed bool, n int64, err error) {
	return false, 0, nil
}
