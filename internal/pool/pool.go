package pool

import (
	"time"
)

type entry struct {
	containerID string
	lastUsed    time.Time
}

type ContainerPool struct {
	idle chan entry
}

func (p *ContainerPool) Acquire() (string, bool) {
	select {
	case e := <-p.idle:
		return e.containerID, true // got a warm container
	default:
		return "", false // pool empty, caller must spin up new one
	}
}

func (p *ContainerPool) Release(containerID string) bool {
	select {
	case p.idle <- entry{containerID, time.Now()}:
		return true // returned to pool
	default:
		return false // pool full, caller must kill
	}
}
