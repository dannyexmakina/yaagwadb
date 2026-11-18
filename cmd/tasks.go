package main

import (
	"fmt"

	"yaagwadb/protocol"

	"github.com/spf13/cobra"
)

const (
	limitDaemonDefault = 0     // without --limit, the daemon's default count
	allDefault         = false // without --all, only running and pending tasks
	allQueues          = ""    // without --queue, all queues
)

const (
	commandTasksDescription = "Show tasks"
	flagAllDescription      = "Include finished tasks"
	flagLimitDescription    = "Number of tasks to show per queue (0 means the daemon default)"
	flagQueueDescription    = "Queue to show (empty means all queues)"
)

const (
	tasksHeader    = "QUEUE\tID\tKIND\tSTATE\tATTEMPTS\tCREATED\tLAST ERROR"
	tasksRowFormat = "%s\t%d\t%s\t%s\t%d\t%s\t%s\n"
)

func newTasksCommand(load configLoader) *cobra.Command {
	var all bool
	var limit uint32
	var queue string
	command := &cobra.Command{
		Use:   commandTasks,
		Short: commandTasksDescription,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			client, err := dial(load)
			if err != nil {
				return err
			}
			defer client.Close()
			response, err := client.ListTasks(command.Context(),
				&protocol.ListTasksRequest{Queue: queue, ActiveOnly: !all, Limit: limit})
			if err != nil {
				return err
			}
			writer := newTableWriter(command.OutOrStdout())
			fmt.Fprintln(writer, tasksHeader)
			for _, task := range response.Tasks {
				fmt.Fprintf(writer, tasksRowFormat, task.Queue, task.Id, task.Kind, task.State, task.Attempts,
					formatTime(task.CreatedAt), task.LastError)
			}
			return writer.Flush()
		},
	}
	command.Flags().BoolVar(&all, flagAll, allDefault, flagAllDescription)
	command.Flags().Uint32Var(&limit, flagLimit, limitDaemonDefault, flagLimitDescription)
	command.Flags().StringVar(&queue, flagQueue, allQueues, flagQueueDescription)
	return command
}
