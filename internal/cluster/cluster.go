package cluster

import "context"

type StateProvider interface {
	PublishStatus(ctx context.Context, status *NodeStatus) error
	GetNodesWithWarmPool(ctx context.Context, funcName string) ([]NodeStatus, error)
	GetHealthyNodes(ctx context.Context) ([]NodeStatus, error)
	Join(ctx context.Context) error
	Leave(ctx context.Context) error
}
