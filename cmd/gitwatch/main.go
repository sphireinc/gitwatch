package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"charm.land/bubbletea/v2"
	"github.com/jusanchez/gitwatch/internal/app"
	"github.com/jusanchez/gitwatch/internal/config"
	"github.com/jusanchez/gitwatch/internal/git"
	"github.com/jusanchez/gitwatch/internal/version"
)

func main() {
	versionFlag := flag.Bool("version", false, "print gitwatch version")
	helpFlag := flag.Bool("help", false, "show help")
	configPath := flag.String("config", "", "configuration file path")
	themeFlag := flag.String("theme", "", "theme: auto, dark, light, or high-contrast")
	motionFlag := flag.String("motion", "", "motion: full, reduced, or off")
	watchFlag := flag.String("watch", "", "watch mode: auto, fs, or poll")
	intervalFlag := flag.Duration("interval", 0, "poll/reconciliation interval")
	configInspect := flag.Bool("config-inspect", false, "print effective configuration and exit")
	configCheck := flag.Bool("config-check", false, "validate configuration and exit")
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
	c, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gitwatch: config: %v\n", err)
		os.Exit(1)
	}
	c = config.ApplyCLI(c, *themeFlag, *motionFlag, *watchFlag, *intervalFlag)
	if err := config.Validate(c); err != nil {
		fmt.Fprintf(os.Stderr, "gitwatch: config: %v\n", err)
		os.Exit(1)
	}
	if *configInspect {
		data, err := config.Inspect(c)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}
	if *configCheck {
		fmt.Println("configuration valid")
		return
	}

	discovery, err := git.Discover(context.Background(), ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "gitwatch: %v\n", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(app.NewRepositoryWithConfig(discovery, c)).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "gitwatch: %v\n", err)
		os.Exit(1)
	}
}
