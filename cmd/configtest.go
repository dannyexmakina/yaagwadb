package main

import (
	"fmt"
	"os"

	"yaagwadb"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// This file holds the checks of the configuration file. Commands run checkConfigFile when loading the configuration,
// the daemon runs yaagwadb.CheckConfig when it opens, and the configtest subcommand runs them all on their own.

const commandConfigTestDescription = "Check the configuration file and exit"

const (
	ConfigFileNotFound    = "yaagwadb: config file not found"
	ConfigFileIsDirectory = "yaagwadb: config file is a directory"
)

const (
	// configFileErrorFormat appends the config file path
	configFileErrorFormat = "%w: %s"
	// configTestPassedFormat is printed when every check passes. %s is the config file path.
	configTestPassedFormat = "%s: configuration OK\n"
)

// checkConfigFile checks that the configuration file exists and is not a directory.
func checkConfigFile(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf(configFileErrorFormat, errors.New(ConfigFileNotFound), path)
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf(configFileErrorFormat, errors.New(ConfigFileIsDirectory), path)
	}
	return nil
}

// checkConfig runs the checks of the daemon and SQL front configuration that the file describes.
func checkConfig(configuration *fileConfig) error {
	settings, err := daemonConfig(configuration)
	if err != nil {
		return err
	}
	if err := yaagwadb.CheckConfig(settings); err != nil {
		return err
	}
	return checkHranaConfig(configuration)
}

func newConfigTestCommand(load configLoader, configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   commandConfigTest,
		Short: commandConfigTestDescription,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			configuration, err := load()
			if err != nil {
				return err
			}
			if err := checkConfig(configuration); err != nil {
				return err
			}
			fmt.Fprintf(command.OutOrStdout(), configTestPassedFormat, *configPath)
			return nil
		},
	}
}
