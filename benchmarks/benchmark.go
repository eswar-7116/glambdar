package main

import (
	"sync"
)

func RunConcurrentBurst(client *Client, funcName string, n int) (coldStarts, warmStarts int) {
	type res struct {
		cold bool
		ok   bool
	}
	ch := make(chan res, n)
	var wg sync.WaitGroup
	for range n {
		wg.Go(func() {
			cold, status := client.MeasureInvokeWithColdStart(funcName)
			ch <- res{cold: cold, ok: status == 200}
		})
	}
	wg.Wait()
	close(ch)
	for r := range ch {
		if r.ok {
			if r.cold {
				coldStarts++
			} else {
				warmStarts++
			}
		}
	}
	return
}
