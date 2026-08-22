// Command releasepack creates deterministic release archives for gitwatch.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sphireinc/git-watch/internal/releasearchive"
)

func main() {
	var output, root, name, format, timestamp string
	flag.StringVar(&output, "output", "", "archive output path")
	flag.StringVar(&root, "root", "", "directory to package")
	flag.StringVar(&name, "name", "", "top-level archive name")
	flag.StringVar(&format, "format", "", "archive format: tar.gz or zip")
	flag.StringVar(&timestamp, "timestamp", "", "normalized RFC3339 timestamp")
	flag.Parse()

	parsedTime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		fail(fmt.Errorf("parse timestamp: %w", err))
	}
	if err := releasearchive.Write(output, root, name, releasearchive.Format(format), parsedTime); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "releasepack: %v\n", err)
	os.Exit(1)
}
