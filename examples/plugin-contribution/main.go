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
		if message.Type == plugin.MessageHandshake {
			response, err := plugin.Encode(plugin.Message{Type: plugin.MessageHandshake, Payload: []byte(`{"api_version":2,"accepted":true,"capabilities":["table"]}`)})
			if err != nil {
				continue
			}
			if _, err := os.Stdout.Write(response); err != nil {
				return err
			}
			continue
		}
		contribution, err := plugin.NewContribution("example.health", plugin.Contribution{
			SchemaVersion: plugin.APIVersion2,
			Kind:          "table",
			Title:         "Example health",
			Columns:       []plugin.TableColumn{{ID: "state", Title: "State"}},
			Rows:          []plugin.TableRow{{"state": "ready"}},
			ReadOnly:      true,
		})
		if err != nil {
			continue
		}
		response, err := plugin.Encode(contribution)
		if err != nil {
			continue
		}
		if _, err := os.Stdout.Write(response); err != nil {
			return err
		}
	}
	return scanner.Err()
}
