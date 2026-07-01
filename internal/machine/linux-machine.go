//go:build linux

package machine

import "syscall"

type LinuxMachine struct{}

func NewLinuxMachine() *LinuxMachine {
	return &LinuxMachine{}
}

func (m *LinuxMachine) Shutdown() {
	syscall.Sync()
	syscall.Reboot(syscall.LINUX_REBOOT_CMD_POWER_OFF)
}
