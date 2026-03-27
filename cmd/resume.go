package cmd

import (
	"bufio"
	"fmt"
	"os"
	"sync"
)

var (
	resumeFile   string
	resumeSet    map[string]bool
	resumeWriter *os.File
	resumeMu     sync.Mutex
)

func loadResumeState(path string) {
	resumeSet = make(map[string]bool)
	if path == "" {
		return
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		logger.Log.Errorf("Could not open resume file for reading: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			resumeSet[line] = true
		}
	}
	logger.Log.Infof("Loaded %d already-tested entries from resume file", len(resumeSet))
}

func initResumeWriter(path string) {
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Log.Errorf("Could not open resume file for writing: %v", err)
		return
	}
	resumeWriter = f
}

func shouldSkip(key string) bool {
	if resumeFile == "" {
		return false
	}
	resumeMu.Lock()
	defer resumeMu.Unlock()
	return resumeSet[key]
}

func markTested(key string) {
	if resumeWriter == nil {
		return
	}
	resumeMu.Lock()
	defer resumeMu.Unlock()
	resumeSet[key] = true
	resumeWriter.WriteString(fmt.Sprintf("%s\n", key))
	resumeWriter.Sync()
}

func closeResumeFile() {
	if resumeWriter != nil {
		resumeWriter.Close()
	}
}
