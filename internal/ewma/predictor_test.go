package ewma

import (
	"math"
	"sync"
	"testing"
)

func TestNewTrafficPredictor_ValidAlpha(t *testing.T) {
	alphas := []float64{0.01, 0.1, 0.2, 0.5, 0.99}
	for _, a := range alphas {
		p, err := NewTrafficPredictor(a)
		if err != nil {
			t.Errorf("NewTrafficPredictor(%f) returned error: %v", a, err)
		}
		if p == nil {
			t.Errorf("NewTrafficPredictor(%f) returned nil", a)
		}
	}
}

func TestNewTrafficPredictor_InvalidAlpha(t *testing.T) {
	invalids := []float64{0, -0.5, 1.0, 1.5, -1.0}
	for _, a := range invalids {
		p, err := NewTrafficPredictor(a)
		if err == nil {
			t.Errorf("NewTrafficPredictor(%f) should have returned error", a)
		}
		if p != nil {
			t.Errorf("NewTrafficPredictor(%f) should return nil on error", a)
		}
	}
}

func TestUpdate_FirstCall_SetsEWMA(t *testing.T) {
	p, err := NewTrafficPredictor(0.2)
	if err != nil {
		t.Fatal(err)
	}

	result := p.Update(100)
	if result != 100.0 {
		t.Errorf("first Update should return the input exactly, got %f", result)
	}
}

func TestUpdate_StableTraffic(t *testing.T) {
	p, err := NewTrafficPredictor(0.2)
	if err != nil {
		t.Fatal(err)
	}

	for range 50 {
		p.Update(50)
	}
	result := p.Update(50)

	if math.Abs(result-50) > 0.1 {
		t.Errorf("EWMA should converge to 50 under stable traffic, got %f", result)
	}
}

func TestUpdate_TrafficSpike(t *testing.T) {
	p, err := NewTrafficPredictor(0.2)
	if err != nil {
		t.Fatal(err)
	}

	// Baseline: 20
	for range 10 {
		p.Update(20)
	}
	baseline := p.Update(20)

	// Spike to 200
	spike1 := p.Update(200)
	spike2 := p.Update(200)

	if spike1 <= baseline {
		t.Errorf("EWMA should increase after spike: baseline=%f, after spike=%f", baseline, spike1)
	}
	if spike2 <= spike1 {
		t.Errorf("EWMA should continue increasing: spike1=%f, spike2=%f", spike1, spike2)
	}
}

func TestUpdate_TrafficDrop(t *testing.T) {
	p, err := NewTrafficPredictor(0.2)
	if err != nil {
		t.Fatal(err)
	}

	// High baseline
	for range 10 {
		p.Update(100)
	}
	high := p.Update(100)

	// Drop to 10
	drop1 := p.Update(10)
	drop2 := p.Update(10)

	if drop1 >= high {
		t.Errorf("EWMA should decrease after drop: high=%f, drop1=%f", high, drop1)
	}
	if drop2 >= drop1 {
		t.Errorf("EWMA should continue decreasing: drop1=%f, drop2=%f", drop1, drop2)
	}
}

func TestUpdate_ZeroTraffic(t *testing.T) {
	p, err := NewTrafficPredictor(0.2)
	if err != nil {
		t.Fatal(err)
	}

	// Some traffic
	for range 5 {
		p.Update(50)
	}

	// Zero traffic
	var last float64
	for range 20 {
		last = p.Update(0)
	}

	if last > 1.0 {
		t.Errorf("EWMA should decay toward 0 with sustained zero traffic, got %f", last)
	}
}

func TestUpdate_ConcurrentSafe(t *testing.T) {
	p, err := NewTrafficPredictor(0.2)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func(val float64) {
			defer wg.Done()
			p.Update(val)
		}(float64(i))
	}
	wg.Wait()

	result := p.Update(50)
	if math.IsNaN(result) || math.IsInf(result, 0) {
		t.Errorf("concurrent updates produced invalid EWMA: %f", result)
	}
}
