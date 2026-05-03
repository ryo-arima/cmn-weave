package model

import (
	"net"
	"time"
)

// NodeStatus represents the operational state of a registered agent node.
type NodeStatus string

const (
	NodeStatusUnknown  NodeStatus = "UNKNOWN"
	NodeStatusOnline   NodeStatus = "ONLINE"
	NodeStatusOffline  NodeStatus = "OFFLINE"
	NodeStatusDraining NodeStatus = "DRAINING"
)

// Node represents a registered agent host managed in the database.
//
// A node corresponds to a single agent process; the server discovers it
// by the gRPC endpoint and tracks its liveness via periodic health checks.
type Node struct {
	// ID is a server-assigned UUIDv4 that uniquely identifies the node.
	ID string

	// IPv4 is the default (primary) IPv4 address of the node.
	// This is the address the server uses to reach the agent's gRPC port.
	IPv4 net.IP

	// IPv6 is the optional primary IPv6 address of the node.
	IPv6 net.IP

	// Hostname is the FQDN or short name reported by the agent at registration.
	Hostname string

	// Status is the current operational state of the node.
	Status NodeStatus

	// CreatedAt is the time this node record was first inserted.
	CreatedAt time.Time

	// UpdatedAt is the time this node record was last modified.
	UpdatedAt time.Time

	// DeletedAt is the soft-delete timestamp.
	// A non-zero value means the node has been deregistered.
	DeletedAt *time.Time
}
