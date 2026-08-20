package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"charm.land/bubbletea/v2"
	"github.com/jusanchez/gitwatch/internal/app"
	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/version"
)

func main() {
	versionFlag := flag.Bool("version", false, "print gitwatch version")
	helpFlag := flag.Bool("help", false, "show help")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "gitwatch — interactive Git worktree dashboard\n\nUsage: gitwatch [options]\n\nOptions:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *versionFlag {
		fmt.Println(version.String())
		return
	}
	if *helpFlag {
		flag.Usage()
		return
	}

	discovery, err := git.Discover(context.Background(), ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gitwatch: %v\n", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(app.NewRepository(discovery)).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gitwatch: %v\n", err)
		os.Exit(1)
	}
}
