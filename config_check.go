package yaagwadb

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// This file holds the checks of the daemon configuration. Open and RestoreForce run them at startup,
// and the configtest subcommand runs them on their own.

const (
	// unsetDataDirectory means Config.DataDirectory is not set. The data directory is required.
	unsetDataDirectory = ""
)

const (
	configErrorFormatDataDirectoryRequired = "%w: DataDirectory is required"
	// configErrorFormatInvalidDatabaseFileName appends the database file name
	configErrorFormatInvalidDatabaseFileName = "%w: DatabaseFileName %q must be a file name without a directory"
	// configErrorFormatGracePeriodNotLonger appends the grace period and the index backup interval
	configErrorFormatGracePeriodNotLonger = "%w: LogicalDeleteGracePeriod (%s) must be longer than IndexArchiveInterval (%s) when object storage is configured"
)

// CheckConfig checks the daemon configuration. Zero-valued fields are checked as their defaults.
func CheckConfig(config Config) error {
	config = config.withDefaults()
	if config.DataDirectory == unsetDataDirectory {
		return fmt.Errorf(configErrorFormatDataDirectoryRequired, errors.New(InvalidArgument))
	}
	if !validDatabaseFileName(config.DatabaseFileName) {
		return fmt.Errorf(configErrorFormatInvalidDatabaseFileName, errors.New(InvalidArgument), config.DatabaseFileName)
	}
	// With object storage, physical deletion must not DELETE a content from object storage while the latest index backup
	// still refers to it. Otherwise a restore from that backup would lack the content.
	if config.ObjectStorage != nil && config.LogicalDeleteGracePeriod <= config.IndexArchiveInterval {
		return fmt.Errorf(configErrorFormatGracePeriodNotLonger, errors.New(InvalidArgument), config.LogicalDeleteGracePeriod, config.IndexArchiveInterval)
	}
	return nil
}

func validDatabaseFileName(name string) bool {
	return name != currentDirectoryName && name != parentDirectoryName && name != "" && !strings.ContainsAny(name, string(filepath.Separator)+"/")
}
