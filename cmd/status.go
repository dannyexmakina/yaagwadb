package main

import (
	"fmt"

	"yaagwadb"

	"github.com/spf13/cobra"
)

const commandStatusDescription = "Show the daemon status"

const (
	statusSummaryFormat  = "file data:          %d\nlogically deleted:  %d\n\n"
	statusQueueHeader    = "QUEUE\tPENDING\tRUNNING\tFAILED"
	statusQueueRowFormat = "%s\t%d\t%d\t%d\n"
)

func dial(load configLoader) (*yaagwadb.Client, error) {
	configuration, err := load()
	if err != nil {
		return nil, err
	}
	return yaagwadb.Dial(configuration.Socket)
}

func newStatusCommand(load configLoader) *cobra.Command {
	return &cobra.Command{
		Use:   commandStatus,
		Short: commandStatusDescription,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, err := dial(load)
			if err != nil {
				return err
			}
			defer client.Close()
			status, err := client.Status(command.Context())
			if err != nil {
				return err
			}
			output := command.OutOrStdout()
			fmt.Fprintf(output, statusSummaryFormat,
				status.FileData, status.LogicallyDeleted)
			writer := newTableWriter(output)
			fmt.Fprintln(writer, statusQueueHeader)
			for _, queue := range status.Queues {
				fmt.Fprintf(writer, statusQueueRowFormat, queue.Queue, queue.Pending, queue.Running, queue.Failed)
			}
			return writer.Flush()
		},
	}
}
