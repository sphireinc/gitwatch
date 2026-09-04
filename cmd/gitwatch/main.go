package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"charm.land/bubbletea/v2"
	"github.com/sphireinc/git-watch/internal/app"
	"github.com/sphireinc/git-watch/internal/config"
	"github.com/sphireinc/git-watch/internal/diagnostics"
	"github.com/sphireinc/git-watch/internal/git"
	"github.com/sphireinc/git-watch/internal/version"
)

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "internal" && os.Args[2] == "sequence-editor" {
		if err := git.HandleSequenceEditor(os.Args[3:], os.Environ()); err != nil {
			fmt.Fprintf(os.Stderr, "gitwatch: sequence editor: %v\n", err)
			os.Exit(1)
		}
		return
	}
	versionFlag := flag.Bool("version", false, "print gitwatch version")
	helpFlag := flag.Bool("help", false, "show help")
	configPath := flag.String("config", "", "configuration file path")
	themeFlag := flag.String("theme", "", "theme: auto, dark, light, or high-contrast")
	motionFlag := flag.String("motion", "", "motion: full, reduced, or off")
	watchFlag := flag.String("watch", "", "watch mode: auto, fs, or poll")
	intervalFlag := flag.Duration("interval", 0, "poll/reconciliation interval")
	groupFlag := flag.String("group", "", "open the repository dashboard filtered to a configured group")
	profileFlag := flag.String("profile", "", "select a named keymap profile")
	commitTreeFlag := flag.Bool("with-commit-tree", false, "show the bounded commit tree in the status pane")
	configInspect := flag.Bool("config-inspect", false, "print effective configuration and exit")
	configCheck := flag.Bool("config-check", false, "validate configuration and exit")
	configMigrationDryRun := flag.Bool("config-migration-dry-run", false, "explain configuration migration without writing the source file")
	diagnosticsFlag := flag.Bool("diagnostics", false, "print sanitized local diagnostics and exit")
	supportBundle := flag.String("support-bundle", "", "write a sanitized private diagnostics bundle and exit")
	flag.Usage = func() {
		if _, err := fmt.Fprint(flag.CommandLine.Output(), "gitwatch — interactive Git worktree dashboard\n\nUsage: gitwatch [options]\n\nOptions:\n"); err != nil {
			return
		}
		flag.PrintDefaults()
	}
	flag.Parse()
	if *configMigrationDryRun {
		var err error
		path := *configPath
		if path == "" {
			path, err = config.Path()
			if err != nil {
				fmt.Fprintf(os.Stderr, "gitwatch: config migration: %v\n", err)
				os.Exit(1)
			}
		}
		plan, err := config.PlanMigrationFile(path)
		if os.IsNotExist(err) {
			plan = config.MigrationPlan{SourceVersion: config.CurrentVersion, TargetVersion: config.CurrentVersion}
		} else if err != nil {
			fmt.Fprintf(os.Stderr, "gitwatch: config migration: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(config.FormatMigrationPlan(plan))
		return
	}
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
	if *commitTreeFlag {
		c.ShowCommitTree = true
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
		if *diagnosticsFlag || *supportBundle != "" {
			report := diagnostics.Build(version.Version, c, git.Discovery{}, "startup", err)
			if *supportBundle != "" {
				if writeErr := diagnostics.WriteBundle(*supportBundle, report); writeErr != nil {
					fmt.Fprintf(os.Stderr, "gitwatch: diagnostics: %v\n", writeErr)
					os.Exit(1)
				}
			} else {
				data, _ := json.MarshalIndent(report, "", "  ")
				fmt.Println(string(data))
			}
			return
		}
		fmt.Fprintf(os.Stderr, "gitwatch: %v\n", err)
		os.Exit(1)
	}
	if *diagnosticsFlag || *supportBundle != "" {
		report := diagnostics.Build(version.Version, c, discovery, "startup", nil)
		if *supportBundle != "" {
			if err := diagnostics.WriteBundle(*supportBundle, report); err != nil {
				fmt.Fprintf(os.Stderr, "gitwatch: diagnostics: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(*supportBundle)
		} else {
			data, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(data))
		}
		return
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
