package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"yaagwadb/assets"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	commandInstallDescription    = "Write an example configuration file and the systemd units. Existing files are left unchanged"
	flagUnitDirectory            = "unit-directory"
	flagUnitDirectoryDescription = "Directory to put the systemd units in"
)

const (
	// defaultUnitDirectory is where administrators place systemd units.
	defaultUnitDirectory = "/etc/systemd/system"
	// configDirectoryPermission and configFilePermission are the permissions of the config file and its directory.
	// The config file may hold object storage credentials, so others may not read it.
	configDirectoryPermission fs.FileMode = 0o755
	configFilePermission      fs.FileMode = 0o640
	// unitFilePermission is the permission of systemd units.
	unitFilePermission fs.FileMode = 0o644
)

const (
	installWroteFormat = "wrote %s\n"
	installKeptFormat  = "kept existing %s\n"
)

func newInstallCommand(configPath *string) *cobra.Command {
	var unitDirectory string
	command := &cobra.Command{
		Use:   commandInstall,
		Short: commandInstallDescription,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			written, err := installConfig(*configPath)
			if err != nil {
				return err
			}
			reportInstall(command, *configPath, written)
			for _, name := range unitFileNames {
				path := filepath.Join(unitDirectory, name)
				written, err := installUnit(name, path)
				if err != nil {
					return err
				}
				reportInstall(command, path, written)
			}
			return nil
		},
	}
	command.Flags().StringVar(&unitDirectory, flagUnitDirectory, defaultUnitDirectory, flagUnitDirectoryDescription)
	return command
}

func reportInstall(command *cobra.Command, path string, written bool) {
	if written {
		fmt.Fprintf(command.OutOrStdout(), installWroteFormat, path)
	} else {
		fmt.Fprintf(command.OutOrStdout(), installKeptFormat, path)
	}
}

var unitFileNames = []string{assets.DaemonUnitFileName, assets.MountUnitFileName, assets.HranaUnitFileName}

// installConfig writes the example config file only if path does not exist. It reports whether it wrote the file.
func installConfig(path string) (bool, error) {
	exists, err := fileExists(path)
	if err != nil || exists {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), configDirectoryPermission); err != nil {
		return false, err
	}
	return true, writeAsset(assets.ConfigFileName, path, configFilePermission)
}

// installUnit writes the systemd unit name only if path does not exist, so that an administrator's edits survive
// another install. It reports whether it wrote the file.
func installUnit(name, path string) (bool, error) {
	exists, err := fileExists(path)
	if err != nil || exists {
		return false, err
	}
	return true, writeAsset(name, path, unitFilePermission)
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func writeAsset(name, path string, permission fs.FileMode) error {
	data, err := assets.Files.ReadFile(name)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, permission); err != nil {
		return err
	}
	// WriteFile does not apply permissions to existing files, so set them again
	return os.Chmod(path, permission)
}
