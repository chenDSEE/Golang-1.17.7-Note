// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements sysSocket for platforms that provide a fast path for
// setting SetNonblock and CloseOnExec.

//go:build dragonfly || freebsd || illumos || linux || netbsd || openbsd
// +build dragonfly freebsd illumos linux netbsd openbsd

package net

import (
	"internal/poll"
	"os"
	"syscall"
)

// Wrapper around the socket system call that marks the returned file
// descriptor as nonblocking and close-on-exec.
func sysSocket(family, sotype, proto int) (int, error) {
	// SOCK_NONBLOCK: 看，虽然 read、write 的时候，是 block，但实际设置的确实非阻塞
	/* SOCK_CLOEXEC(close on exec() call): 
	 * Note that the use of this flag is essential in some multithreaded programs.
	 * 这个 fd 我在 fork 子进程后执行 exec() 时就关闭，避免父进程重启后，无法再次监听这个端口（因为子进程占用着）
	 * 这就意味着，这个 socket fd 是不能被子进程继承的。
	 * 既可以是是避免 fd leak，也可以是权限控制，避免高权限进程 open 的 fd，被低权限的子进程使用
	 */
	s, err := socketFunc(family, sotype|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC, proto)
	// On Linux the SOCK_NONBLOCK and SOCK_CLOEXEC flags were
	// introduced in 2.6.27 kernel and on FreeBSD both flags were
	// introduced in 10 kernel. If we get an EINVAL error on Linux
	// or EPROTONOSUPPORT error on FreeBSD, fall back to using
	// socket without them.
	switch err {
	case nil:
		return s, nil
	default:
		return -1, os.NewSyscallError("socket", err)
	case syscall.EPROTONOSUPPORT, syscall.EINVAL:
		// 兼容不支持 SOCK_NONBLOCK、SOCK_CLOEXEC 旧版本系统
	}

	// See ../syscall/exec_unix.go for description of ForkLock.
	syscall.ForkLock.RLock()
	s, err = socketFunc(family, sotype, proto)
	if err == nil {
		syscall.CloseOnExec(s)
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return -1, os.NewSyscallError("socket", err)
	}
	if err = syscall.SetNonblock(s, true); err != nil {
		poll.CloseFunc(s)
		return -1, os.NewSyscallError("setnonblock", err)
	}
	return s, nil
}
