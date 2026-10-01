package cluster

import "time"

type NodeStatus struct {
	NodeID   string                `json:"nodeId"`
	Address  string                `json:"address"` // host:grpcPort
	Pools    map[string]PoolStatus `json:"pools"`   // funcName -> poolStatus
	LastBeat time.Time             `json:"lastBeat"`
	Capacity ResourceCapacity      `json:"capacity"`
}

type PoolStatus struct {
	IdleCount      int   `json:"idleCount"`
	ActiveCount    int   `json:"activeCount"`
	MaxConcurrency int32 `json:"maxConcurrency"`
}

type ResourceCapacity struct {
	CPUAvailable    float64 `json:"cpuAvailable"`
	MemoryAvailable int64   `json:"memoryAvailable"`
}
