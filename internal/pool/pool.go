package pool

import (
	"time"
)

type entry struct {
	containerID string
	socketPath  string
	lastUsed    time.Time
}

type ContainerPool struct {
	idle chan entry
}

func (p *ContainerPool) Acquire() (containerID string, socketPath string, warm bool) {
	select {
	case e := <-p.idle:
		return e.containerID, e.socketPath, true // got a warm container
	default:
		return "", "", false // pool empty, caller must spin up new one
	}
}

func (p *ContainerPool) Release(containerID, socketPath string) bool {
	select {
	case p.idle <- entry{containerID, socketPath, time.Now()}:
		return true // returned to pool
	default:
		return false // pool full, caller must kill
	}
}
