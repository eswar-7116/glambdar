package ewma

import (
	"fmt"
	"sync"
)

type TrafficPredictor struct {
	baseAlpha   float64
	alpha       float64
	ewma        float64
	initialized bool
	mu          sync.Mutex
}

func NewTrafficPredictor(baseAlpha float64) (*TrafficPredictor, error) {
	if baseAlpha <= 0 || baseAlpha >= 1 {
		return nil, fmt.Errorf("alpha must be in (0, 1), got %f", baseAlpha)
	}
	return &TrafficPredictor{
		baseAlpha: baseAlpha,
		alpha:     baseAlpha,
	}, nil
}

func (p *TrafficPredictor) Update(count float64) float64 {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		p.ewma = count
		p.initialized = true
		return p.ewma
	}

	deviation := abs(count - p.ewma)
	sensitivity := deviation / (p.ewma + 1)
	p.alpha = clamp(p.baseAlpha+(1-p.baseAlpha)*sensitivity, 0.05, 0.95)

	p.ewma = p.alpha*count + (1-p.alpha)*p.ewma
	return p.ewma
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
