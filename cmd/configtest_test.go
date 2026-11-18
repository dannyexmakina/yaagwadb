package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	// name of a config file that does not exist
	configTestMissingName = "missing.toml"
	// name of the config file written in the test
	configTestName = "yaagwadb.toml"
	// name of the data directory set in the config file written in the test
	configTestDataDirectoryName = "data"
	// config file with object storage whose grace period is shorter than the index backup interval (%q is the data directory)
	configTestShortGracePeriodFormat = "data_directory = %q\n" +
		"[object_storage]\nkind = \"s3\"\nbucket = \"bucket\"\nregion = \"us-east-1\"\n" +
		"[daemon]\nindex_archive_interval = \"1h\"\nlogical_delete_grace_period = \"1m\"\n"
	// config file with only the data directory (%q is the data directory)
	configTestMinimalFormat = "data_directory = %q\n"
	// text expected in the error of a grace period not longer than the index backup interval
	configTestGracePeriodErrorText = "LogicalDeleteGracePeriod"
	// failure message format when the error does not contain the expected text
	configTestWantErrorTextFormat = "want %q in the error, got %v"
)

func writeConfigTestFile(t *testing.T, format string) string {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, configTestName)
	content := fmt.Sprintf(format, filepath.Join(directory, configTestDataDirectoryName))
	if err := os.WriteFile(path, []byte(content), configFilePermission); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertErrorContains(t *testing.T, err error, text string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), text) {
		t.Fatalf(configTestWantErrorTextFormat, text, err)
	}
}

func TestCheckConfigFile(t *testing.T) {
	directory := t.TempDir()
	missing := filepath.Join(directory, configTestMissingName)
	assertErrorContains(t, checkConfigFile(missing), ConfigFileNotFound)
	assertErrorContains(t, checkConfigFile(directory), ConfigFileIsDirectory)
	_, err := loadConfig(missing)
	assertErrorContains(t, err, ConfigFileNotFound)
}

func TestCheckConfig(t *testing.T) {
	configuration, err := loadConfig(writeConfigTestFile(t, configTestShortGracePeriodFormat))
	if err != nil {
		t.Fatal(err)
	}
	assertErrorContains(t, checkConfig(configuration), configTestGracePeriodErrorText)

	configuration, err = loadConfig(writeConfigTestFile(t, configTestMinimalFormat))
	if err != nil {
		t.Fatal(err)
	}
	if err := checkConfig(configuration); err != nil {
		t.Fatal(err)
	}
}
