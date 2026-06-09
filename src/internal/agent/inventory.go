package agent

import (
	"context"
	"time"

	"zhiliu/internal/management"
)

type InventoryReporter interface {
	Report(ctx context.Context, clusterID string, agent management.ClusterAgent) (management.ReportAcceleratorInventoryRequest, bool, error)
}

type NoopInventoryReporter struct{}

type FakeInventoryReporter struct {
	Now func() time.Time
}

func (NoopInventoryReporter) Report(context.Context, string, management.ClusterAgent) (management.ReportAcceleratorInventoryRequest, bool, error) {
	return management.ReportAcceleratorInventoryRequest{}, false, nil
}

func (r FakeInventoryReporter) Report(_ context.Context, clusterID string, agent management.ClusterAgent) (management.ReportAcceleratorInventoryRequest, bool, error) {
	now := time.Now
	if r.Now != nil {
		now = r.Now
	}
	observedAt := now().UTC()
	return management.ReportAcceleratorInventoryRequest{
		AgentID:       agent.ID,
		SchemaVersion: "accelerator-inventory/v1alpha1",
		ObservedAt:    observedAt,
		Nodes: []management.AcceleratorInventoryNode{
			{
				Name: "fake-node-1",
				Labels: map[string]string{
					"nvidia.com/gpu.product":  "NVIDIA-H200-SXM",
					"inference.platform/fake": "true",
				},
				Capacity: map[string]string{
					"nvidia.com/gpu": "8",
				},
				Allocatable: map[string]string{
					"nvidia.com/gpu": "8",
				},
				Accelerators: []management.AcceleratorInventoryAccelerator{
					{
						Vendor:      "nvidia",
						Product:     "NVIDIA H200 SXM",
						DeviceCount: 8,
						MemoryMiB:   143360,
						VendorDetails: map[string]string{
							"source": "fake",
						},
					},
				},
				ObservedAt: observedAt,
			},
		},
		ProbeStatuses: []management.AcceleratorInventoryProbe{
			{Name: "fake-inventory", Status: "ok", Message: "synthetic local development inventory"},
		},
		CollectionMetadata: map[string]string{
			"mode":      "fake",
			"clusterId": clusterID,
		},
	}, true, nil
}
