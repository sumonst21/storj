// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package overlay

import (
	"context"

	"storj.io/storj/satellite/internalpb"
)

// Inspector is a RPC service for inspecting overlay internals.
//
// architecture: Endpoint
type Inspector struct {
	internalpb.DRPCOverlayInspectorUnimplementedServer
	service *Service
}

// NewInspector creates an Inspector.
func NewInspector(service *Service) *Inspector {
	return &Inspector{service: service}
}

// CountNodes returns the number of nodes in the overlay.
func (srv *Inspector) CountNodes(ctx context.Context, req *internalpb.CountNodesRequest) (_ *internalpb.CountNodesResponse, err error) {
	defer mon.Task()(&ctx)(&err)
	overlayKeys, err := srv.service.Inspect(ctx)
	if err != nil {
		return nil, err
	}

	return &internalpb.CountNodesResponse{
		Count: int64(len(overlayKeys)),
	}, nil
}
