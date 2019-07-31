// +build arm7tdmi

package runtime

import (
	"device/arm"
	"unsafe"
)

type timeUnit int64

const tickMicros = 1

func putchar(c byte) {
	// dummy, TODO
}

//go:extern _sbss
var _sbss unsafe.Pointer

//go:extern _ebss
var _ebss unsafe.Pointer

//go:extern _sdata
var _sdata unsafe.Pointer

//go:extern _sidata
var _sidata unsafe.Pointer

//go:extern _edata
var _edata unsafe.Pointer

// Entry point for Go. Initialize all packages and call main.main().
//go:export main
func main() {
	// Initialize .data and .bss sections.
	preinit()

	// Run initializers of all packages.
	initAll()

	// Compiler-generated call to main.main().
	callMain()
}

func preinit() {
	// Initialize .bss: zero-initialized global variables.
	ptr := uintptr(unsafe.Pointer(&_sbss))
	for ptr != uintptr(unsafe.Pointer(&_ebss)) {
		*(*uint32)(unsafe.Pointer(ptr)) = 0
		ptr += 4
	}

	// Initialize .data: global variables initialized from flash.
	src := uintptr(unsafe.Pointer(&_sidata))
	dst := uintptr(unsafe.Pointer(&_sdata))
	for dst != uintptr(unsafe.Pointer(&_edata)) {
		*(*uint32)(unsafe.Pointer(dst)) = *(*uint32)(unsafe.Pointer(src))
		dst += 4
		src += 4
	}
}

func ticks() timeUnit {
	// TODO
	return 0
}

const asyncScheduler = false

func sleepTicks(d timeUnit) {
	// TODO
}

func abort() {
	// TODO
	for {
	}
}

func getCurrentStackPointer() uintptr {
	return arm.ReadRegister("sp")
}

// Implement memset for LLVM and compiler-rt.
//go:export memset
func libc_memset(ptr unsafe.Pointer, c byte, size uintptr) {
	for i := uintptr(0); i < size; i++ {
		*(*byte)(unsafe.Pointer(uintptr(ptr) + i)) = c
	}
}

// Implement memmove for LLVM and compiler-rt.
//go:export memmove
func libc_memmove(dst, src unsafe.Pointer, size uintptr) {
	memmove(dst, src, size)
}
