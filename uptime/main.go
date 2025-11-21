package main

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	ntdll = syscall.NewLazyDLL("ntdll.dll")
	procNtQuerySystemInformation = ntdll.NewProc("NtQuerySystemInformation")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	procFileTimeToSystemTime = kernel32.NewProc("FileTimeToSystemTime")
	procFileTimeToLocalFileTime = kernel32.NewProc("FileTimeToLocalFileTime")
)

type SystemTimeOfDayInformation struct {
	BootTime          int64
	CurrentTime       int64
	TimeZoneBias      int64
	CurrentTimeZoneId uint32
}

type SystemTime struct {
	Year         uint16
	Month        uint16
	DayOfWeek    uint16
	Day          uint16
	Hour         uint16
	Minute       uint16
	Second       uint16
	Milliseconds uint16
}

func getSystemUpTime() (time.Time, error) {
	var sysTimeInfo SystemTimeOfDayInformation

	ret, _, _ := procNtQuerySystemInformation.Call(
		3,
		uintptr(unsafe.Pointer(&sysTimeInfo)),
		uintptr(unsafe.Sizeof(sysTimeInfo)),
		0,
	)

	if ret != 0 {
		return time.Time{}, fmt.Errorf("NtQuerySystemInformation failed with status 0x%X", ret)
	}

	// Convert FILETIME (bootTime) to local FILETIME
	var localFileTime int64
	ret, _, _ = procFileTimeToLocalFileTime.Call(
		uintptr(unsafe.Pointer(&sysTimeInfo.BootTime)),
		uintptr(unsafe.Pointer(&localFileTime)),
	)
	if ret == 0 {
		return time.Time{}, fmt.Errorf("FileTimeToLocalFileTime failed")
	}

	// Convert FILETIME to SYSTEMTIME
	var sysTime SystemTime
	ret, _, _ = procFileTimeToSystemTime.Call(
		uintptr(unsafe.Pointer(&localFileTime)),
		uintptr(unsafe.Pointer(&sysTime)),
	)
	if ret == 0 {
		return time.Time{}, fmt.Errorf("FileTimeToSystemTime failed")
	}

	// Convert SYSTEMTIME to time.Time
	return time.Date(
		int(sysTime.Year),
		time.Month(sysTime.Month),
		int(sysTime.Day),
		int(sysTime.Hour),
		int(sysTime.Minute),
		int(sysTime.Second),
		int(sysTime.Milliseconds)*int(time.Millisecond),
		time.Local,
	), nil
}

func main() {
	bootTime, err := getSystemUpTime()
	if err != nil {
		fmt.Printf("Error getting system uptime: %v\n", err)
		return
	}

	fmt.Printf("System Uptime: %02d/%02d/%04d %02d:%02d:%02d\n",
		bootTime.Month(),
		bootTime.Day(),
		bootTime.Year(),
		bootTime.Hour(),
		bootTime.Minute(),
		bootTime.Second())
}
