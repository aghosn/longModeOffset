package main

import (
	"fmt"
)

var (
	code = []uint8{
		0xba, 0xf8, 0x03, /* mov $0x3f8, %dx */
		0x00, 0xd8, /* add %bl, %al */
		0x04, '0', /* add $'0', %al */
		0xee,       /* out %al, (%dx) */
		0xb0, '\n', /* mov $'\n', %al */
		0xee, /* out %al, (%dx) */
		0xf4, /* hlt */
	}
)

func main() {
	fmt.Println("Trying stuff")
	vm := &fakeVm{}
	vmInit(vm)
	mapAreas(vm)
	vcpuInit(vm)
	initPageTables(vm)
	initPageTables2(vm)
	initSRegs(vm)
	initURegs(vm)
	runVM(vm)
}
