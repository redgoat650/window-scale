package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"syscall"
	"unsafe"
)

var (
	windowToMaxOpt = flag.String("windowName", "Visual Studio Code", "Pass the name of the window you wish to maximize over multiple monitors")
	allMode        = flag.Bool("all", false, "Set this flag to resize all 'windowName' matches")
	borderLen      = flag.Int("border", 0, "Size buffer on the border of the scaled window for fine tuning")
)

var (
	user32                  = syscall.MustLoadDLL("user32.dll")
	procEnumWindows         = user32.MustFindProc("EnumWindows")
	procGetWindowTextW      = user32.MustFindProc("GetWindowTextW")
	procMoveWindow          = user32.MustFindProc("MoveWindow")
	procGetDesktopWindow    = user32.MustFindProc("GetDesktopWindow")
	procGetWindowRect       = user32.MustFindProc("GetWindowRect")
	procMonitorFromPoint    = user32.MustFindProc("MonitorFromPoint")
	procGetMonitorInfo      = user32.MustFindProc("GetMonitorInfoA")
	procEnumDisplayMonitors = user32.MustFindProc("EnumDisplayMonitors")
)

func EnumWindows(enumFunc uintptr, lparam uintptr) (err error) {
	r1, _, e1 := syscall.Syscall(procEnumWindows.Addr(), 2, uintptr(enumFunc), uintptr(lparam), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func GetWindowText(hwnd syscall.Handle, str *uint16, maxCount int32) (len int32, err error) {
	r0, _, e1 := syscall.Syscall(procGetWindowTextW.Addr(), 3, uintptr(hwnd), uintptr(unsafe.Pointer(str)), uintptr(maxCount))
	len = int32(r0)
	if len == 0 {
		if e1 != 0 {
			err = error(e1)
		} else {
			err = syscall.EINVAL
		}
	}
	return
}

func EnumDisplayMonitors(enumFunc uintptr, lparam uintptr) (err error) {

	r1, _, e1 := syscall.Syscall6(procEnumDisplayMonitors.Addr(), 4, uintptr(0), uintptr(0), uintptr(enumFunc), uintptr(lparam), uintptr(0), uintptr(0))
	if r1 == 0 {
		if e1 != 0 {
			err = error(e1)
		}
	}
	return
}

func FindMonitors() (resultRect rect, err error) {

	resultRect = rect{}
	largestHeight := int32(0)

	cb := syscall.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
		monInfo, monInfoErr := GetMonitorInfo(h)
		if monInfoErr != nil {
			//ignore error
			return 1
		}
		height := monInfo.rcWork.bottom - monInfo.rcWork.top
		// width := monInfo.rcWork.right - monInfo.rcWork.left

		if height > largestHeight {
			largestHeight = height
			resultRect.top = monInfo.rcWork.top
			resultRect.bottom = monInfo.rcWork.bottom
			resultRect.left = 0
			resultRect.right = 0
		}

		if height == largestHeight {
			if monInfo.rcWork.right > resultRect.right {
				resultRect.right = monInfo.rcWork.right
			}
			if monInfo.rcWork.left < resultRect.left {
				resultRect.left = monInfo.rcWork.left
			}
		}

		fmt.Println(monInfo)
		fmt.Println(resultRect)
		return 1
	})

	EnumDisplayMonitors(cb, 0)
	return
}

func FindWindow(title string) ([]syscall.Handle, error) {
	var hwnd []syscall.Handle
	cb := syscall.NewCallback(func(h syscall.Handle, p uintptr) uintptr {
		b := make([]uint16, 200)
		_, err := GetWindowText(h, &b[0], int32(len(b)))
		if err != nil {
			// ignore the error
			return 1 // continue enumeration
		}
		fmt.Println(syscall.UTF16ToString(b))
		if strings.Contains(syscall.UTF16ToString(b), title) {
			// note the window
			hwnd = append(hwnd, h)
			if !*allMode {
				return 0 // stop enumeration
			}
		}
		return 1 // continue enumeration
	})
	EnumWindows(cb, 0)
	if len(hwnd) == 0 {
		return nil, fmt.Errorf("No window with title '%s' found", title)
	}
	return hwnd, nil
}

func MoveWindow(hwnd syscall.Handle, x int, y int, nWidth int, nHeight int, bRepaint bool) (err error) {
	_, _, e1 := syscall.Syscall6(procMoveWindow.Addr(), 6, uintptr(hwnd), uintptr(x), uintptr(y), uintptr(nWidth), uintptr(nHeight), 1)

	if e1 != 0 {
		err = error(e1)
	}

	return
}

func GetDesktopWindow() (hwnd syscall.Handle, err error) {
	_, _, e1 := syscall.Syscall(procGetDesktopWindow.Addr(), 1, uintptr(hwnd), uintptr(0), uintptr(0))
	if e1 != 0 {
		err = error(e1)
	}
	return
}

type rect struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

func GetWindowRect(hwnd syscall.Handle) (lprect rect, err error) {

	lprect = rect{}

	r0, _, e1 := syscall.Syscall(procGetWindowRect.Addr(), 2, uintptr(hwnd), uintptr(unsafe.Pointer(&lprect)), uintptr(0))
	if r0 == 0 {
		fmt.Println("blab")
	}
	if e1 != 0 {
		fmt.Println("Flab")
		err = error(e1)
	}
	return
}

type point struct {
	x int32
	y int32
}

func MonitorFromPoint(pt point, dwFlags uint16) (hmonitor syscall.Handle, err error) {
	fmt.Println(pt)
	h, _, e1 := syscall.Syscall(procMonitorFromPoint.Addr(), 2, uintptr(unsafe.Pointer(&pt)), uintptr(dwFlags), uintptr(0))
	if e1 != 0 {
		err = error(e1)
	}
	fmt.Println(h)
	hmonitor = syscall.Handle(h)
	return
}

type monitorinfo struct {
	cbsize    uint16
	rcMonitor rect
	rcWork    rect
	dwFlags   uint16
}

func GetMonitorInfo(hmonitor syscall.Handle) (lpmonitorInfo monitorinfo, err error) {
	lpmonitorInfo = monitorinfo{}
	lpmonitorInfo.cbsize = uint16(unsafe.Sizeof(lpmonitorInfo))

	_, _, e1 := syscall.Syscall(procGetMonitorInfo.Addr(), 2, uintptr(hmonitor), uintptr(unsafe.Pointer(&lpmonitorInfo)), uintptr(0))
	if e1 != 0 {
		err = error(e1)
	}
	return
}

func main() {
	flag.Parse()
	title := *windowToMaxOpt
	h, err := FindWindow(title)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found '%s' window(s):\n", title)
	for _, v := range h {
		fmt.Printf("handle=0x%x\n", v)
	}

	rect, fme := FindMonitors()
	if fme != nil {
		log.Fatal(fme)
	}
	fmt.Println("Resulting rect is", rect)

	for _, v := range h {
		moveErr := MoveWindow(v, int(rect.left)-*borderLen, int(rect.top), int(rect.right-rect.left)+2*(*borderLen), int(rect.bottom-rect.top)+*borderLen, true)
		if moveErr != nil {
			log.Fatal(moveErr)
		}
	}
}
