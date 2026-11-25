//go:build windows

package main

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

const title = "Web-Chromium MCP Setup"

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	gdi32            = syscall.NewLazyDLL("gdi32.dll")
	versionDLL       = syscall.NewLazyDLL("version.dll")
	messageBoxW      = user32.NewProc("MessageBoxW")
	createWindow     = user32.NewProc("CreateWindowExW")
	destroyWindow    = user32.NewProc("DestroyWindow")
	setWindowText    = user32.NewProc("SetWindowTextW")
	peekMessage      = user32.NewProc("PeekMessageW")
	dispatchMessage  = user32.NewProc("DispatchMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	systemMetrics    = user32.NewProc("GetSystemMetrics")
	sendMessage      = user32.NewProc("SendMessageW")
	stockObject      = gdi32.NewProc("GetStockObject")
	versionSize      = versionDLL.NewProc("GetFileVersionInfoSizeW")
	versionInfo      = versionDLL.NewProc("GetFileVersionInfoW")
	versionQuery     = versionDLL.NewProc("VerQueryValueW")
)

func wide(text string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(text)
	return p
}

func dialog(text string, flags uintptr) uintptr {
	result, _, _ := messageBoxW.Call(0, uintptr(unsafe.Pointer(wide(text))), uintptr(unsafe.Pointer(wide(title))), flags)
	return result
}

func browserMajor(path string) uint32 {
	length, _, _ := versionSize.Call(uintptr(unsafe.Pointer(wide(path))), 0)
	if length == 0 || length > 1024*1024 {
		return 0
	}
	data := make([]byte, length)
	ok, _, _ := versionInfo.Call(uintptr(unsafe.Pointer(wide(path))), 0, length, uintptr(unsafe.Pointer(&data[0])))
	if ok == 0 {
		return 0
	}
	var fixed *[4]uint32
	var size uint32
	ok, _, _ = versionQuery.Call(uintptr(unsafe.Pointer(&data[0])), uintptr(unsafe.Pointer(wide("\\"))), uintptr(unsafe.Pointer(&fixed)), uintptr(unsafe.Pointer(&size)))
	var major uint32
	if ok != 0 && size >= 16 && fixed[0] == 0xfeef04bd {
		major = fixed[2] >> 16
	}
	runtime.KeepAlive(data)
	return major
}

// Keep the native progress window responsive while WinGet and Codex run.
func showProgress(install func(func(string)) error) error {
	const width, height = 510, 145
	screenWidth, _, _ := systemMetrics.Call(0)
	screenHeight, _, _ := systemMetrics.Call(1)
	window, _, _ := createWindow.Call(0, uintptr(unsafe.Pointer(wide("STATIC"))), uintptr(unsafe.Pointer(wide(title))), 0x10c00000,
		(screenWidth-width)/2, (screenHeight-height)/2, width, height, 0, 0, 0, 0)
	if window == 0 {
		return install(func(string) {})
	}
	label, _, _ := createWindow.Call(0, uintptr(unsafe.Pointer(wide("STATIC"))), uintptr(unsafe.Pointer(wide("Preparing setup…"))), 0x50000000,
		22, 25, 460, 60, window, 0, 0, 0)
	font, _, _ := stockObject.Call(17)
	sendMessage.Call(label, 0x0030, font, 1)
	defer destroyWindow.Call(window)

	updates := make(chan string, 8)
	done := make(chan error, 1)
	go func() { done <- install(func(text string) { updates <- text }) }()
	ticker := time.NewTicker(30 * time.Millisecond)
	defer ticker.Stop()

	// MSG contains pointers and padding. The array covers the full struct on
	// both 64-bit Windows targets and keeps its address aligned.
	var msg [8]uintptr
	for {
		select {
		case err := <-done:
			return err
		case text := <-updates:
			setWindowText.Call(label, uintptr(unsafe.Pointer(wide(text))))
		case <-ticker.C:
		}
		for {
			ok, _, _ := peekMessage.Call(uintptr(unsafe.Pointer(&msg[0])), 0, 0, 0, 1)
			if ok == 0 {
				break
			}
			translateMessage.Call(uintptr(unsafe.Pointer(&msg[0])))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&msg[0])))
		}
		runtime.KeepAlive(&msg)
	}
}
