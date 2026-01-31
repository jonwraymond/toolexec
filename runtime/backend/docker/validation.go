package docker

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
	for i, m := range s.Mounts {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("mount[%d]: %w", i, err)
		}
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

// Validate checks Mount for required fields.
func (m Mount) Validate() error {
	if m.Target == "" {
		return errors.New("target is required")
	}
	switch m.Type {
	case MountTypeBind:
		if m.Source == "" {
			return errors.New("source is required for bind mounts")
		}
	case MountTypeVolume:
		if m.Source == "" {
			return errors.New("source is required for volume mounts")
		}
	case MountTypeTmpfs:
		// tmpfs doesn't require source
	case "":
		return errors.New("mount type is required")
	default:
		return fmt.Errorf("unknown mount type: %s", m.Type)
	}
	return nil
}
