// إنشاء الاختصار بواجهة ويندوز الرسمية IShellLinkW (نفس اللي بيستخدمه ويندوز نفسه) من غير PowerShell
// جميع الحقوق محفوظة للمصمم والمطور - م/ أسامه حسن بدري - 01145060007
package main

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	pCoInit   = ole32.NewProc("CoInitializeEx")
	pCoUninit = ole32.NewProc("CoUninitialize")
	pCoCreate = ole32.NewProc("CoCreateInstance")
)

var (
	clsidShellLink = guid{0x00021401, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidShellLinkW  = guid{0x000214F9, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidPersistFile = guid{0x0000010B, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
)

func vcall(obj uintptr, idx int, args ...uintptr) uintptr {
	vtbl := *(*uintptr)(unsafe.Pointer(obj))
	fn := *(*uintptr)(unsafe.Pointer(vtbl + uintptr(idx)*unsafe.Sizeof(uintptr(0))))
	r, _, _ := syscall.SyscallN(fn, append([]uintptr{obj}, args...)...)
	return r
}

func ptr(s string) uintptr { return uintptr(unsafe.Pointer(u16(s))) }

func comLnk(path, target, dir, ico, args string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := pCoInit.Call(0, 2) // COINIT_APARTMENTTHREADED
	if hr == 0 || hr == 1 {
		defer pCoUninit.Call()
	}
	var sl uintptr
	r, _, _ := pCoCreate.Call(uintptr(unsafe.Pointer(&clsidShellLink)), 0, 1 /*CLSCTX_INPROC_SERVER*/, uintptr(unsafe.Pointer(&iidShellLinkW)), uintptr(unsafe.Pointer(&sl)))
	if r != 0 || sl == 0 {
		return errors.New("CoCreateInstance")
	}
	defer vcall(sl, 2)
	if vcall(sl, 20, ptr(target)) != 0 { // SetPath
		return errors.New("SetPath")
	}
	vcall(sl, 9, ptr(dir))      // SetWorkingDirectory
	vcall(sl, 7, ptr(appName))  // SetDescription
	vcall(sl, 17, ptr(ico), 0)  // SetIconLocation
	if args != "" {
		vcall(sl, 11, ptr(args)) // SetArguments
	}
	var pf uintptr
	if vcall(sl, 0, uintptr(unsafe.Pointer(&iidPersistFile)), uintptr(unsafe.Pointer(&pf))) != 0 || pf == 0 {
		return errors.New("IPersistFile")
	}
	defer vcall(pf, 2)
	if vcall(pf, 6, ptr(path), 1) != 0 { // Save
		return errors.New("Save")
	}
	return nil
}
