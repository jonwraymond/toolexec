package firecracker

import (
	"errors"
)

// Validate checks MicroVMSpec for errors before execution.
func (s MicroVMSpec) Validate() error {
	if s.Config.KernelPath == "" {
		return errors.New("kernelPath is required")
	}
	if s.Config.RootfsPath == "" {
		return errors.New("rootfsPath is required")
	}
	if s.Resources.VCPUCount <= 0 {
		return errors.New("vcpuCount must be positive")
	}
	if s.Resources.MemSizeMB <= 0 {
		return errors.New("memSizeMB must be positive")
	}
	return nil
}
