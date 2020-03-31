package main

import (
	"gosb/vtx/old"
)

const (
	_KVM_DRIVER_PATH = "/dev/kvm"
	_PageSize        = 0x1000
)

type fakeVm struct {
	sysFd      int
	fd         int
	vcpu       *fakevCPU
	sregs      kvm_sregs
	regs       kvm_regs
	mem        uintptr
	pageTables uintptr
	bonus      uintptr
}

// runData is the run structure. This may be mapped for synchronous register
// access (although that doesn't appear to be supported by my kernel at least).
//
// This mirrors kvm_run.
type fakeRunData struct {
	requestInterruptWindow uint8
	_                      [7]uint8

	exitReason                 uint32
	readyForInterruptInjection uint8
	ifFlag                     uint8
	_                          [2]uint8

	cr8      uint64
	apicBase uint64

	// This is the union data for exits. Interpretation depends entirely on
	// the exitReason above (see vCPU code for more information).
	data [32]uint64
}

// userMemoryRegion is a region of physical memory.
//
// This mirrors kvm_memory_region.
type userMemoryRegion struct {
	slot          uint32
	flags         uint32
	guestPhysAddr uint64
	memorySize    uint64
	userspaceAddr uint64
}

type fakevCPU struct {
	fd  int
	run *fakeRunData
}

type kvm_segment struct {
	base     uint64
	limit    uint32
	selector uint16
	typ      uint8
	present  uint8
	dpl      uint8
	db       uint8
	s        uint8
	l        uint8
	g        uint8
	avl      uint8
	unusable uint8
	padding  uint8
}

type kvm_dtable struct {
	base    uint64
	limit   uint16
	padding [3]uint16
}

/* for KVM_GET_SREGS and KVM_SET_SREGS */
type kvm_sregs struct {
	/* out (KVM_GET_SREGS) / in (KVM_SET_SREGS) */
	cs               kvm_segment
	ds               kvm_segment
	es               kvm_segment
	fs               kvm_segment
	gs               kvm_segment
	ss               kvm_segment
	tr               kvm_segment
	ldt              kvm_segment
	gdt              kvm_dtable
	idt              kvm_dtable
	cr0              uint64
	cr2              uint64
	cr3              uint64
	cr4              uint64
	cr8              uint64
	efer             uint64
	apic_base        uint64
	interrupt_bitmap [(old.KVM_NR_INTERRUPTS + 63) / 64]uint64
}

/* for KVM_GET_REGS and KVM_SET_REGS */
type kvm_regs struct {
	/* out (KVM_GET_REGS) / in (KVM_SET_REGS) */
	rax    uint64
	rbx    uint64
	rcx    uint64
	rdx    uint64
	rsi    uint64
	rdi    uint64
	rsp    uint64
	rbp    uint64
	r8     uint64
	r9     uint64
	r10    uint64
	r11    uint64
	r12    uint64
	r13    uint64
	r14    uint64
	r15    uint64
	rip    uint64
	rflags uint64
}
