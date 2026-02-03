package proxmox

import "context"

// APIClient defines the Proxmox API interactions needed by the backend.
type APIClient interface {
	Status(ctx context.Context, node string, vmid int) (LXCStatus, error)
	Start(ctx context.Context, node string, vmid int) error
	Stop(ctx context.Context, node string, vmid int) error
}

// LXCStatus describes a container status response.
type LXCStatus struct {
	Status string `json:"status"`
}
