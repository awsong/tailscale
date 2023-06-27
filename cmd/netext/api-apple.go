/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2018-2019 Jason A. Donenfeld <Jason@zx2c4.com>. All Rights Reserved.
 */

package main

// #include <stdio.h>
// #include <stdlib.h>
// #include <sys/types.h>
// static void callLogger(void *func, void *ctx, int level, const char *msg)
// {
// 	((void(*)(void *, int, const char *))func)(ctx, level, msg);
// }
// void SwiftIntfSet(const char *localAddrs, const char *routes);
import "C"

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"unsafe"

	"github.com/tailscale/wireguard-go/device"
	"github.com/tailscale/wireguard-go/tun"
	"golang.org/x/sys/unix"
	"tailscale.com/net/dns"
	"tailscale.com/types/logger"
	"tailscale.com/wgengine/router"
)

var loggerFunc unsafe.Pointer
var loggerCtx unsafe.Pointer

type CLogger int

func cstring(s string) *C.char {
	b, err := unix.BytePtrFromString(s)
	if err != nil {
		b := [1]C.char{}
		return &b[0]
	}
	return (*C.char)(unsafe.Pointer(b))
}

func (l CLogger) Printf(format string, args ...interface{}) {
	if uintptr(loggerFunc) == 0 {
		return
	}
	C.callLogger(loggerFunc, loggerCtx, C.int(l), cstring(fmt.Sprintf(format, args...)))
}

func init() {
	signals := make(chan os.Signal)
	signal.Notify(signals, unix.SIGUSR2)
	go func() {
		buf := make([]byte, os.Getpagesize())
		for {
			select {
			case <-signals:
				n := runtime.Stack(buf, true)
				buf[n] = 0
				if uintptr(loggerFunc) != 0 {
					C.callLogger(loggerFunc, loggerCtx, 0, (*C.char)(unsafe.Pointer(&buf[0])))
				}
			}
		}
	}()
}

//export wgSetLogger
func wgSetLogger(context, loggerFn uintptr) {
	loggerCtx = unsafe.Pointer(context)
	loggerFunc = unsafe.Pointer(loggerFn)
}

var b *backend

//export wgTurnOn
func wgTurnOn(path *C.char, tunFd int32) int32 {
	deviceLogger := &device.Logger{
		Verbosef: CLogger(0).Printf,
		Errorf:   CLogger(1).Printf,
	}

	dupTunFd, err := unix.Dup(int(tunFd))
	if err != nil {
		deviceLogger.Errorf("Unable to dup tun fd: %v", err)
		return -1
	}

	err = unix.SetNonblock(dupTunFd, true)
	if err != nil {
		deviceLogger.Errorf("Unable to set tun fd as non blocking: %v", err)
		unix.Close(dupTunFd)
		return -1
	}
	f := os.NewFile(uintptr(dupTunFd), "/dev/tun")
	tunDev, err := tun.CreateTUNFromFile(f, 0)
	if err != nil {
		deviceLogger.Errorf("Unable to create new tun device from fd: %v", err)
		unix.Close(dupTunFd)
		return -1
	}

	tstunNew = func(logf logger.Logf, tunName string) (tun.Device, string, error) {
		return tunDev, f.Name(), nil
	}

	b, err = newBackend(tunDev, C.GoString(path), deviceLogger.Errorf, setBoth)
	if err != nil {
		deviceLogger.Errorf("Unable to create backend: %v", err)
		return -1
	}
	//go run(C.GoString(path), deviceLogger.Errorf)
	return 0
}

//export wgTurnOff
func wgTurnOff() {

}

func setBoth(r *router.Config, d *dns.OSConfig) error {
	localAddrs := r.LocalAddrs
	routes := r.Routes

	// Convert your Go slices to a form that can be passed to Swift.
	// This will depend on what your Swift function expects.
	// For the sake of this example, let's assume your Swift function takes two arrays of strings.
	var localAddrsStrs []string
	var routesStrs []string

	for _, addr := range localAddrs {
		localAddrsStrs = append(localAddrsStrs, addr.String())
	}

	for _, route := range routes {
		routesStrs = append(routesStrs, route.String())
	}

	// Convert the Go string slices to C arrays.
	// Again, this is a simplification and you'll need to adjust this to match your actual requirements.
	localAddrsCArray := C.CString(strings.Join(localAddrsStrs, ","))
	routesCArray := C.CString(strings.Join(routesStrs, ","))

	// Call the Swift function.
	C.SwiftIntfSet(localAddrsCArray, routesCArray)
	return nil
}
