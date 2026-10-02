package api

import (
	"github.com/eswar-7116/glambdar/v3/internal/cluster"
	"github.com/eswar-7116/glambdar/v3/internal/controller"
)

var (
	controllerMode bool
	clusterRouter  *cluster.Router
	grpcPool       *controller.GRPCClientPool
	stateProvider  cluster.StateProvider
)

func SetControllerMode(router *cluster.Router, pool *controller.GRPCClientPool, state cluster.StateProvider) {
	controllerMode = true
	clusterRouter = router
	grpcPool = pool
	stateProvider = state
}
