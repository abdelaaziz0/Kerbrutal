package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	jsonOutput   bool
	jsonOutputMu sync.Mutex
)

type ResultEvent struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Username  string `json:"username"`
	Domain    string `json:"domain"`
	Password    string `json:"password,omitempty"`
	Message     string `json:"message,omitempty"`
	Hash        string `json:"hash,omitempty"`
	HashcatMode int    `json:"hashcat_mode,omitempty"`
	EType       int32  `json:"etype,omitempty"`
}

func emitJSON(eventType, username, password, message, hash string, hashcatMode int, etype int32) {
	if !jsonOutput {
		return
	}
	event := ResultEvent{
		Timestamp:   time.Now().Format(time.RFC3339),
		Type:        eventType,
		Username:    username,
		Domain:      domain,
		Password:    password,
		Message:     message,
		Hash:        hash,
		HashcatMode: hashcatMode,
		EType:       etype,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	jsonOutputMu.Lock()
	defer jsonOutputMu.Unlock()
	fmt.Fprintf(os.Stdout, "\r\033[2K%s\n", string(data))
}
