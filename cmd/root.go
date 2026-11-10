package cmd

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/ayn2op/arikawa/v3/utils/ws"
	"github.com/ayn2op/discordo/internal/cli"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/ayn2op/discordo/internal/ui/root"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

func Run() error {
	// Subcommands are handled without starting the TUI, so the CLI never opens
	// a gateway connection or a terminal screen.
	if len(os.Args) > 1 && cli.IsCommand(os.Args[1]) {
		return cli.Run(os.Args[1:])
	}

	configPath := flag.String("config-path", config.DefaultPath(), "path of the configuration file")
	logPath := flag.String("log-path", logger.DefaultPath(), "path of the log file")
	logLevel := flag.String("log-level", "info", "log level")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s\n", buildVersion())
		return nil
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(*logLevel)); err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	ws.EnableRawEvents = level == slog.LevelDebug

	logFile, err := logger.Load(*logPath, level)
	if err != nil {
		return fmt.Errorf("failed to load logger: %w", err)
	}
	defer logFile.Close() //nolint:errcheck

	exe, _ := os.Executable()
	slog.Info("starting discordo", "version", buildVersion(), "exe", exe)

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return fmt.Errorf("failed to init screen: %w", err)
	}

	if cfg.Mouse {
		screen.EnableMouse()
	}
	screen.EnablePaste()
	if cfg.Notifications.WhenUnfocused {
		screen.EnableFocus()
	}

	return tview.NewApplication(root.NewModel(cfg), tview.WithScreen(screen)).Run()
}
