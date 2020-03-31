package main

import (
	"fmt"
	"gosb/commons"
	"gosb/vtx/old"
	"os"
	"syscall"
	"unsafe"
)

// works for  134217727
// doesnt for 134217730
// trying for 134217730

// winner work 134217727
// winner doesn't 134217728

const (
	bonusOffset = uintptr(20 * _PageSize)
	_mmapHint   = uintptr(0x100000)
)

func vmInit(vm *fakeVm) {
	sysFd, err := os.OpenFile(_KVM_DRIVER_PATH, syscall.O_RDWR, 0)
	if err != nil {
		panic("Damn kvm")
	}
	vm.sysFd = int(sysFd.Fd())
	fd, errno := commons.Ioctl(vm.sysFd, old.KVM_CREATE_VM, 0)
	if errno != 0 {
		panic("KVM_CREATE_VM")
	}
	vm.fd = fd
	/*_, errno = commons.Ioctl(vm.fd, old.KVM_SET_TSS_ADDR, uintptr(commons.ReservedMemory-3*0x1000))
	if errno != 0 {
		panic("Damn tss")
	}*/

}

func mapAreas(vm *fakeVm) {
	var (
		errno syscall.Errno
	)

	// This is for the code.
	vm.mem, errno = commons.Mmap(_mmapHint, _PageSize, syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS, -1, 0)
	if errno != 0 {
		panic("Damn mmap")
	}

	// This is for the page table.
	vm.pageTables, errno = commons.Mmap(0, 20*_PageSize, syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS, -1, 0)
	if errno != 0 {
		panic("Damn mmap 2")
	}

	// This is a bonus area that we map extra just to see if it works.
	vm.bonus, errno = commons.Mmap(0, _PageSize, syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS, -1, 0)
	if errno != 0 {
		panic("Damn mmap 2")
	}
	// Register the pagetables at the beginning.
	ptregs := userMemoryRegion{
		slot:          0,
		flags:         0,
		guestPhysAddr: 0,
		memorySize:    uint64(20 * _PageSize),
		userspaceAddr: uint64(vm.pageTables),
	}

	// Map the bonus area after the page tables.
	bonusRegs := userMemoryRegion{
		slot:          1,
		flags:         0,
		guestPhysAddr: uint64(bonusOffset),
		memorySize:    _PageSize,
		userspaceAddr: uint64(vm.bonus),
	}

	// Register the code at the right spot.
	memregs := userMemoryRegion{
		slot:          2,
		flags:         0,
		guestPhysAddr: uint64(vm.mem),
		memorySize:    uint64(_PageSize),
		userspaceAddr: uint64(vm.mem),
	}

	// do the ioctls.
	_, errno = commons.Ioctl(vm.fd, old.KVM_SET_USER_MEMORY_REGION, uintptr(unsafe.Pointer(&memregs)))
	if errno != 0 {
		panic("Damn user memory region")
	}
	_, errno = commons.Ioctl(vm.fd, old.KVM_SET_USER_MEMORY_REGION, uintptr(unsafe.Pointer(&ptregs)))
	if errno != 0 {
		panic("Damn user memory region")
	}
	_, errno = commons.Ioctl(vm.fd, old.KVM_SET_USER_MEMORY_REGION, uintptr(unsafe.Pointer(&bonusRegs)))
	if errno != 0 {
		panic("Damn user memory region")
	}
}

func vcpuInit(vm *fakeVm) {
	var (
		errno syscall.Errno
	)
	vm.vcpu = &fakevCPU{}
	vm.vcpu.fd, errno = commons.Ioctl(vm.fd, old.KVM_CREATE_VCPU, 0)
	if errno != 0 {
		panic("KVM_CREATE_VCPU")
	}
	mmap_size, errno := commons.Ioctl(vm.sysFd, old.KVM_GET_VCPU_MMAP_SIZE, 0)
	if errno != 0 {
		panic("Damn vcpu mmap size")
	}
	vcpuMem, err := commons.Mmap(0, uintptr(mmap_size), syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED, vm.vcpu.fd, 0)
	if err != 0 {
		panic("Damn mmap run")
	}
	vm.vcpu.run = (*fakeRunData)(unsafe.Pointer(vcpuMem))
}

type fakePte = [512]uintptr

// initPageTables direct mapping HVA = GPA = GVA for code.
func initPageTables(vm *fakeVm) {
	// we need to init the page tables here.
	// We do it by hand since we're mapping only one page.
	pml4Addr := uintptr(0x2000)
	pml4 := (*fakePte)(unsafe.Pointer(vm.pageTables + pml4Addr))
	pml4Idx := old.PDX(vm.mem, old.LVL_PML4)

	pdptAddr := uintptr(0x3000)
	pdpt := (*fakePte)(unsafe.Pointer(vm.pageTables + pdptAddr))
	pdptIdx := old.PDX(vm.mem, old.LVL_PDPTE)

	pdAddr := uintptr(0x4000)
	pd := (*fakePte)(unsafe.Pointer(vm.pageTables + pdAddr))
	pdIdx := old.PDX(vm.mem, old.LVL_PDE)

	pteAddr := uintptr(0x5000)
	pte := (*fakePte)(unsafe.Pointer(vm.pageTables + pteAddr))
	pteIdx := old.PDX(vm.mem, old.LVL_PTE)

	pml4[pml4Idx] = old.PTE_P | old.PTE_W | old.PTE_U | pdptAddr
	pdpt[pdptIdx] = old.PTE_P | old.PTE_W | old.PTE_U | pdAddr
	pd[pdIdx] = old.PTE_P | old.PTE_W | old.PTE_U | pteAddr
	pte[pteIdx] = old.PTE_P | old.PTE_W | old.PTE_U | (bonusOffset)
	fmt.Printf("memory address %x\n", vm.mem)
	fmt.Printf("pml4: %d, pdpt: %d, pd: %d, pte: %d\n", pml4Idx, pdptIdx, pdIdx, pteIdx)

	// Check we copied correctly
	expected := (*uintptr)(unsafe.Pointer(vm.pageTables + pml4Addr + uintptr(pml4Idx)*unsafe.Sizeof(uintptr(0))))
	if *expected == 0 {
		panic("pml4 entry is 0")
	}
	if *expected != pml4[pml4Idx] {
		panic("Not the same entries")
	}
	if expected != &pml4[pml4Idx] {
		panic("Not the same addresses")
	}

	// copy the code in mem and in bonus.
	commons.Memcpy(vm.mem, uintptr(unsafe.Pointer(&code[0])), uintptr(len(code)))
	commons.Memcpy(vm.bonus, uintptr(unsafe.Pointer(&code[0])), uintptr(len(code)))
}

// initPageTables2 indirect mapping of code.
func initPageTables2(vm *fakeVm) {
	// we need to init the page tables here.
	// We do it by hand since we're mapping only one page.
	pml4Addr := uintptr(0x2000)
	pml4 := (*uintptr)(unsafe.Pointer(vm.pageTables + pml4Addr))

	pdptAddr := uintptr(0x7000)
	pdpt := (*uintptr)(unsafe.Pointer(vm.pageTables + pdptAddr))

	pdAddr := uintptr(0x8000)
	pd := (*uintptr)(unsafe.Pointer(vm.pageTables + pdAddr))

	pteAddr := uintptr(0x9000)
	pte := (*uintptr)(unsafe.Pointer(vm.pageTables + pteAddr))

	*pml4 = old.PTE_P | old.PTE_W | old.PTE_U | pdptAddr
	*pdpt = old.PTE_P | old.PTE_W | old.PTE_U | pdAddr
	*pd = old.PTE_P | old.PTE_W | old.PTE_U | pteAddr
	*pte = old.PTE_P | old.PTE_W | old.PTE_U | vm.mem //(bonusOffset)

	commons.Memcpy(uintptr(vm.pageTables+0x000), uintptr(unsafe.Pointer(&code[0])), uintptr(len(code)))
}

func initSRegs(vm *fakeVm) {
	// First get the sregs
	_, err := commons.Ioctl(vm.vcpu.fd, old.KVM_GET_SREGS, uintptr(unsafe.Pointer(&vm.sregs)))
	if err != 0 {
		panic("Damn get sregs")
	}
	// Then set them up
	vm.sregs.cr3 = uint64(0x2000)
	vm.sregs.cr4 = old.CR4_PAE
	vm.sregs.cr0 = old.CR0_PE | old.CR0_MP | old.CR0_ET | old.CR0_NE | old.CR0_WP | old.CR0_AM | old.CR0_PG
	vm.sregs.efer = old.EFER_LME | old.EFER_LMA

	seg := kvm_segment{
		base:     0,
		limit:    0xffffffff,
		selector: 1 << 3,
		present:  1,
		typ:      11,
		dpl:      0,
		db:       0,
		s:        1,
		l:        1,
		g:        1,
	}
	vm.sregs.cs = seg
	seg.typ = 3
	seg.selector = 2 << 3
	vm.sregs.ds = seg
	vm.sregs.es = seg
	vm.sregs.fs = seg
	vm.sregs.gs = seg
	vm.sregs.ss = seg

	_, err = commons.Ioctl(vm.vcpu.fd, old.KVM_SET_SREGS, uintptr(unsafe.Pointer(&vm.sregs)))
	if err != 0 {
		panic("Damn set sregs")
	}
}

func initURegs(vm *fakeVm) {
	vm.regs.rflags = 2
	vm.regs.rip = uint64(vm.mem)
	vm.regs.rsp = 0x00 //2 << 20
	_, err := commons.Ioctl(vm.vcpu.fd, old.KVM_SET_REGS, uintptr(unsafe.Pointer(&vm.regs)))
	if err != 0 {
		panic("Damn set regs")
	}
}

func runVM(vm *fakeVm) {
	// Run that bitch
	_, err := commons.Ioctl(vm.vcpu.fd, old.KVM_RUN, 0)
	if err != 0 {
		fmt.Println("err ", err)
		panic("Die bitch")
	}
	fmt.Println("Success run with exit  ", vm.vcpu.run.exitReason)
}
