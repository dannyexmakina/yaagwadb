package main

import (
	"yaagwadb"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	RestoreWithoutForce = "yaagwadb: restore moves the local data aside; pass " + flagPrefix + flagForce + " to proceed"
)

// forceDefault is the value when --force is omitted. Without it, no restore happens.
const forceDefault = false

const (
	commandRestoreDescription = "Keep broken local data under another name and restore from object storage"
	flagForceDescription      = "Keep the local index and file data under another name and restore"
)

func newRestoreCommand(load configLoader) *cobra.Command {
	var force bool
	command := &cobra.Command{
		Use:   commandRestore + usageSeparator + flagPrefix + flagForce,
		Short: commandRestoreDescription,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if !force {
				return errors.New(RestoreWithoutForce)
			}
			configuration, err := load()
			if err != nil {
				return err
			}
			settings, err := daemonConfig(configuration)
			if err != nil {
				return err
			}
			daemon, err := yaagwadb.RestoreForce(settings)
			if err != nil {
				return err
			}
			return daemon.Close()
		},
	}
	command.Flags().BoolVar(&force, flagForce, forceDefault, flagForceDescription)
	return command
}
