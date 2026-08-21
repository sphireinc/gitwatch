package main

import (
	"bufio"
	"fmt"
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
		payload := []byte(fmt.Sprintf(`{"id":%q,"title":"Run %s"}`, message.ID, message.Type))
		response, err := plugin.Encode(plugin.Message{Type: plugin.MessageCommand, ID: message.ID, Payload: payload})
		if err == nil {
			_, _ = os.Stdout.Write(response)
		}
	}
}
