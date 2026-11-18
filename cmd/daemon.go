package main

import (
	"yaagwadb"

	"github.com/spf13/cobra"
)

const commandDaemonDescription = "Run as the daemon (gRPC server)"

const (
	logDaemonStarted    = "daemon started"
	logDaemonStopping   = "daemon stopping"
	logKeySocket        = "socket"
	logKeyDataDirectory = "data"
)

func newDaemonCommand(load configLoader) *cobra.Command {
	return &cobra.Command{
		Use:   commandDaemon,
		Short: commandDaemonDescription,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			configuration, err := load()
			if err != nil {
				return err
			}
			settings, err := daemonConfig(configuration)
			if err != nil {
				return err
			}
			daemon, err := yaagwadb.Open(settings)
			if err != nil {
				return err
			}
			server, err := yaagwadb.ListenGRPC(daemon, configuration.Socket)
			if err != nil {
				daemon.Close()
				return err
			}
			settings.Logger.Info(logDaemonStarted, logKeySocket, configuration.Socket, logKeyDataDirectory, configuration.DataDirectory)
			waitForSignal()
			settings.Logger.Info(logDaemonStopping)
			server.GracefulStop()
			return daemon.Close()
		},
	}
}
