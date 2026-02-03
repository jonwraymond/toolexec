package containerd

import (
	"errors"
	"fmt"
)

// Validate checks ContainerSpec for errors before execution.
func (s ContainerSpec) Validate() error {
	if s.Image == "" {
		return errors.New("image is required")
	}
	if err := s.Security.Validate(); err != nil {
		return fmt.Errorf("security: %w", err)
	}
	if err := s.Resources.Validate(); err != nil {
		return fmt.Errorf("resources: %w", err)
	}
	return nil
}

// Validate checks SecuritySpec for policy violations.
func (s SecuritySpec) Validate() error {
	if s.Privileged {
		return ErrSecurityViolation
	}
	if s.NetworkMode == "host" {
		return fmt.Errorf("%w: host network not allowed", ErrSecurityViolation)
	}
	return nil
}

// Validate checks ResourceSpec for invalid values.
func (r ResourceSpec) Validate() error {
	if r.MemoryBytes < 0 {
		return errors.New("memory cannot be negative")
	}
	if r.CPUQuota < 0 {
		return errors.New("cpu quota cannot be negative")
	}
	if r.PidsLimit < 0 {
		return errors.New("pids limit cannot be negative")
	}
	if r.DiskBytes < 0 {
		return errors.New("disk limit cannot be negative")
	}
	return nil
}
