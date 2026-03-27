package cmd

import (
	"bufio"
	"math/rand"
	"os"
	"sync"
	"time"
)

var opsecMode bool

func loadAndShuffleFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	shuffleSlice(lines)
	return lines, nil
}

func shuffleSlice(lines []string) {
	rand.Shuffle(len(lines), func(i, j int) {
		lines[i], lines[j] = lines[j], lines[i]
	})
}

var (
	jitterRng   = rand.New(rand.NewSource(time.Now().UnixNano()))
	jitterRngMu sync.Mutex
)

func jitterDelay(baseDelay int) {
	if baseDelay <= 0 {
		return
	}
	jitterRngMu.Lock()
	r := jitterRng.Float64()
	jitterRngMu.Unlock()

	jitter := float64(baseDelay) * 0.3
	actual := float64(baseDelay) + (r*2-1)*jitter
	if actual < 0 {
		actual = 0
	}
	time.Sleep(time.Duration(actual) * time.Millisecond)
}

func applyDelay() {
	if opsecMode {
		jitterDelay(delay)
	} else {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}
