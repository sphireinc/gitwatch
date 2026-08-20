package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var request struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(scanner.Bytes(), &request) != nil {
			continue
		}
		response := map[string]any{"type": "status", "payload": map[string]string{"text": fmt.Sprintf("example plugin received %s", request.Type)}}
		_ = json.NewEncoder(os.Stdout).Encode(response)
	}
}
