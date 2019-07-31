// +build gameboyadvance

package machine

import (
	"image/color"
	"runtime/volatile"
	"unsafe"
)

var Display = FramebufDisplay{(*[160][240]volatile.Register16)(unsafe.Pointer(uintptr(0x06000000)))}

type FramebufDisplay struct {
	port *[160][240]volatile.Register16
}

func (d FramebufDisplay) Size() (x, y int16) {
	return 240, 160
}

func (d FramebufDisplay) SetPixel(x, y int16, c color.RGBA) {
	d.port[y][x].Set(uint16(c.R)&0x1f | uint16(c.G)&0x1f<<5 | uint16(c.B)&0x1f<<10)
}

func (d FramebufDisplay) Display() error {
	// Nothing to do here.
	return nil
}
