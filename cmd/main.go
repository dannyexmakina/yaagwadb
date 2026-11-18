package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

const (
	commandRoot       = "yaagwadb"
	commandConfigTest = "configtest"
	commandDaemon     = "daemon"
	commandHrana      = "hrana"
	commandImport     = "import"
	commandInstall    = "install"
	commandMount      = "mount"
	commandRestore    = "restore"
	commandStatus     = "status"
	commandTasks      = "tasks"
)

const (
	flagConfig      = "config"
	flagConfigShort = "c"
	flagForce       = "force"
	flagAll         = "all"
	flagLimit       = "limit"
	flagQueue       = "queue"
	flagVolume      = "volume"
	flagVolumeShort = "v"
	// flagPrefix precedes flag names in usage and error messages.
	flagPrefix = "--"
)

const (
	// exitFailure is the exit code when a command fails.
	exitFailure = 1
	// signalBuffer is the size of the termination signal channel. signal.Notify drops signals if the receiver is blocked, so buffer one.
	signalBuffer = 1
	// silenceUsage suppresses usage when a command fails. Printing usage for non-argument errors buries the error.
	silenceUsage = true
)

const (
	commandRootDescription = "A daemon that stores files and serves them over gRPC"
	flagConfigDescription  = "Configuration file"
	// usageSeparator separates the command name from arguments and flags in usage.
	usageSeparator = " "
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		os.Exit(exitFailure)
	}
}

func newRootCommand() *cobra.Command {
	var configPath string
	root := &cobra.Command{
		Use:          commandRoot,
		Short:        commandRootDescription,
		SilenceUsage: silenceUsage,
	}
	root.PersistentFlags().StringVarP(&configPath, flagConfig, flagConfigShort, defaultConfigPath, flagConfigDescription)
	load := func() (*fileConfig, error) { return loadConfig(configPath) }
	root.AddCommand(
		newConfigTestCommand(load, &configPath),
		newDaemonCommand(load),
		newHranaCommand(load),
		newImportCommand(load),
		newInstallCommand(&configPath),
		newMountCommand(load),
		newRestoreCommand(load),
		newStatusCommand(load),
		newTasksCommand(load),
	)
	return root
}

func waitForSignal() {
	signals := make(chan os.Signal, signalBuffer)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
}
