package main

import (
	"fmt"

	"yaagwadb"
	"yaagwadb/mount"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	MountPointRequired = "yaagwadb: no mount point specified"
	DaemonUnreachable  = "yaagwadb: cannot connect to the daemon"
)

const (
	mountPointRequiredFormat = "%w. Pass it with %s%s, or set point under [mount] in the configuration file\nUsage: %s\nSee %s for details"
	helpCommandFormat        = "%s %s%s"
	helpFlagName             = "help"
	shortFlagPrefix          = "-"
	daemonUnreachableFormat  = "%w (socket: %s). Check that the daemon is running and that socket in the configuration file is correct. A relative path is resolved against the directory where the command was run: %v"
)

// volumeDefault is the value when -v is omitted. If empty, mount.point from the config is used.
const volumeDefault = ""

const (
	commandMountDescription = "Run as the FUSE mounter (a gRPC client of the daemon)"
	flagVolumeDescription   = "Mount point. If omitted, point under [mount] in the configuration file is used"
)

func newMountCommand(load configLoader) *cobra.Command {
	var volume string
	command := &cobra.Command{
		Use:   commandMount,
		Short: commandMountDescription,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			configuration, err := load()
			if err != nil {
				return err
			}
			mountPoint := configuration.Mount.Point
			if volume != "" {
				mountPoint = volume
			}
			if mountPoint == "" {
				return fmt.Errorf(mountPointRequiredFormat, errors.New(MountPointRequired), shortFlagPrefix, flagVolumeShort, command.UseLine(), helpCommand(command))
			}
			client, err := yaagwadb.Dial(configuration.Socket)
			if err != nil {
				return err
			}
			defer client.Close()
			if _, err := client.Status(command.Context()); err != nil {
				return fmt.Errorf(daemonUnreachableFormat, errors.New(DaemonUnreachable), configuration.Socket, err)
			}
			server, err := mount.Start(client, mountPoint, mount.Options{Debug: configuration.Mount.Debug, AllowOther: configuration.Mount.AllowOther})
			if err != nil {
				return err
			}
			go func() {
				waitForSignal()
				server.Unmount()
			}()
			server.Wait()
			return nil
		},
	}
	command.Flags().StringVarP(&volume, flagVolume, flagVolumeShort, volumeDefault, flagVolumeDescription)
	return command
}

func helpCommand(command *cobra.Command) string {
	return fmt.Sprintf(helpCommandFormat, command.CommandPath(), flagPrefix, helpFlagName)
}
