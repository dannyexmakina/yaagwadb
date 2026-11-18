package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	"yaagwadb"
	"yaagwadb/hrana"

	pkgerrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const commandHranaDescription = "Run as the SQL front (Hrana over HTTP, a gRPC client of the daemon)"

const (
	HranaSizeLimitTooLarge = "yaagwadb: hrana file_data_size_limit is over the gRPC message size"
	HranaSizeLimitRequired = "yaagwadb: hrana file_data_size_limit must be positive"
)

const (
	hranaNetwork               = "tcp"
	logHranaStarted            = "hrana started"
	logHranaStopping           = "hrana stopping"
	logKeyListen               = "listen"
	hranaSizeLimitFormat       = "%w (%d > %d bytes)"
	hranaSizeLimitRequiredForm = "%w (%d)"
	// noFileDataSizeLimit is a size limit that refuses every file data
	noFileDataSizeLimit = 0
)

// grpcMaximumMessageSize is the largest message gRPC receives by default. A whole file data a SELECT returns
// is read through the daemon, so the limit of the SQL front stays within it.
const grpcMaximumMessageSize = 4 << 20

func checkHranaConfig(configuration *fileConfig) error {
	limit := configuration.Hrana.FileDataSizeLimit
	if limit <= noFileDataSizeLimit {
		return fmt.Errorf(hranaSizeLimitRequiredForm, pkgerrors.New(HranaSizeLimitRequired), limit)
	}
	if limit > grpcMaximumMessageSize {
		return fmt.Errorf(hranaSizeLimitFormat, pkgerrors.New(HranaSizeLimitTooLarge), limit, grpcMaximumMessageSize)
	}
	return nil
}

func hranaConfig(configuration *fileConfig) hrana.Config {
	rowLimit := configuration.Daemon.QueryRowLimit
	if rowLimit == 0 {
		rowLimit = yaagwadb.DefaultConfig().QueryRowLimit
	}
	return hrana.Config{
		FileDataSizeLimit: configuration.Hrana.FileDataSizeLimit,
		QueryRowLimit:     rowLimit,
		FileMode:          configuration.Hrana.FileMode,
		UID:               uint32(os.Getuid()),
		GID:               uint32(os.Getgid()),
		Logger:            newLogger(configuration.LogLevel),
	}
}

func newHranaCommand(load configLoader) *cobra.Command {
	return &cobra.Command{
		Use:   commandHrana,
		Short: commandHranaDescription,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			configuration, err := load()
			if err != nil {
				return err
			}
			if err := checkHranaConfig(configuration); err != nil {
				return err
			}
			client, err := yaagwadb.Dial(configuration.Socket)
			if err != nil {
				return err
			}
			defer client.Close()
			if _, err := client.Status(command.Context()); err != nil {
				return fmt.Errorf(daemonUnreachableFormat, pkgerrors.New(DaemonUnreachable), configuration.Socket, err)
			}
			settings := hranaConfig(configuration)
			listener, err := net.Listen(hranaNetwork, configuration.Hrana.Listen)
			if err != nil {
				return err
			}
			server := &http.Server{Handler: hrana.NewServer(client, settings).Handler()}
			go func() {
				waitForSignal()
				settings.Logger.Info(logHranaStopping)
				server.Shutdown(context.Background())
			}()
			settings.Logger.Info(logHranaStarted, logKeyListen, listener.Addr().String(), logKeySocket, configuration.Socket)
			if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		},
	}
}
