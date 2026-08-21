package main

import (
	"bufio"
	"os"

	"github.com/jusanchez/gitwatch/pkg/plugin"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message, err := plugin.Decode(append(scanner.Bytes(), '\n'))
		if err != nil {
			continue
		}
		response, err := plugin.Encode(plugin.Message{Type: plugin.MessageWidget, ID: message.ID, Payload: []byte(`{"id":"example.widget","text":"gitwatch ready"}`)})
		if err == nil {
			_, _ = os.Stdout.Write(response)
		}
	}
}
