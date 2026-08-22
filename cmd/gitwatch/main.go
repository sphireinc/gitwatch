package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"charm.land/bubbletea/v2"
	"github.com/sphireinc/git-watch/internal/app"
	"github.com/sphireinc/git-watch/internal/config"
	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/version"
)

func main() {
	versionFlag := flag.Bool("version", false, "print gitwatch version")
	helpFlag := flag.Bool("help", false, "show help")
	configPath := flag.String("config", "", "configuration file path")
	themeFlag := flag.String("theme", "", "theme: auto, dark, light, or high-contrast")
	motionFlag := flag.String("motion", "", "motion: full, reduced, or off")
	watchFlag := flag.String("watch", "", "watch mode: auto, fs, or poll")
	intervalFlag := flag.Duration("interval", 0, "poll/reconciliation interval")
	groupFlag := flag.String("group", "", "open the repository dashboard filtered to a configured group")
	profileFlag := flag.String("profile", "", "select a named keymap profile")
	configInspect := flag.Bool("config-inspect", false, "print effective configuration and exit")
	configCheck := flag.Bool("config-check", false, "validate configuration and exit")
	flag.Usage = func() {
		if _, err := fmt.Fprint(flag.CommandLine.Output(), "gitwatch — interactive Git worktree dashboard\n\nUsage: gitwatch [options]\n\nOptions:\n"); err != nil {
			return
		}
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
	if *profileFlag != "" {
		c.Profile = *profileFlag
	}
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
	model := app.NewRepositoryWithConfig(discovery, c)
	model.RepositoryGroup = *groupFlag
	finalModel, runErr := tea.NewProgram(model).Run()
	var closeErr error
	switch final := finalModel.(type) {
	case app.Model:
		closeErr = final.Close()
	case *app.Model:
		closeErr = final.Close()
	}
	if runErr != nil {
		fmt.Fprintf(os.Stderr, "gitwatch: %v\n", runErr)
		os.Exit(1)
	}
	if closeErr != nil {
		fmt.Fprintf(os.Stderr, "gitwatch: shutdown: %v\n", closeErr)
		os.Exit(1)
	}
}
