package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/sphireinc/git-watch/pkg/plugin"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		message, err := plugin.Decode(append(scanner.Bytes(), '\n'))
		if err != nil {
			continue
		}
		response, err := plugin.Encode(plugin.Message{Type: plugin.MessageWidget, ID: message.ID, Payload: []byte(`{"id":"example.widget","text":"gitwatch ready"}`)})
		if err != nil {
			continue
		}
		if _, err := os.Stdout.Write(response); err != nil {
			return err
		}
	}
	return scanner.Err()
}
