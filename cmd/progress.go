package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

func countFileLines(path string) int {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() != "" {
			count++
		}
	}
	return count
}

func startProgressTicker(total int) func() {
	if total == 0 {
		return func() {}
	}
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				current := atomic.LoadInt32(&counter)
				valid := atomic.LoadInt32(&successes)
				pct := float64(current) / float64(total) * 100
				fmt.Fprintf(os.Stderr, "\r\033[2K[Progress] %d/%d (%.1f%%) — %d valid so far", current, total, pct, valid)
			}
		}
	}()

	return func() {
		ticker.Stop()
		close(done)
		current := atomic.LoadInt32(&counter)
		valid := atomic.LoadInt32(&successes)
		pct := float64(current) / float64(total) * 100
		fmt.Fprintf(os.Stderr, "\r\033[2K[Progress] %d/%d (%.1f%%) — %d valid\n", current, total, pct, valid)
	}
}
