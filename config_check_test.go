package yaagwadb

import (
	"testing"
	"time"
)

const (
	// Index backup interval in the grace period check test
	testIndexArchiveInterval = time.Hour
	// Grace period shorter than testIndexArchiveInterval
	testShortGracePeriod = time.Minute
	// Database file names that point outside the data directory
	testDatabaseFileNameWithDirectory = "sub/yaagwa.db"
	testDatabaseFileNameParent        = ".."
	testDatabaseFileNameCurrent       = "."
)

func TestCheckConfigRequiresDataDirectory(t *testing.T) {
	if err := CheckConfig(Config{}); !Has(err, InvalidArgument) {
		t.Fatalf(testWantInvalidArgumentFormat, err)
	}
}

func TestCheckConfigRejectsDatabaseFileNameWithDirectory(t *testing.T) {
	for _, name := range []string{testDatabaseFileNameWithDirectory, testDatabaseFileNameParent, testDatabaseFileNameCurrent} {
		config := testConfig(t.TempDir())
		config.DatabaseFileName = name
		if err := CheckConfig(config); !Has(err, InvalidArgument) {
			t.Fatalf(testWantInvalidArgumentFormat, err)
		}
	}
}

// With object storage, a grace period not longer than the index backup interval is rejected.
// Otherwise physical deletion could DELETE a content that the latest index backup still refers to.
func TestGracePeriodMustBeLongerThanIndexArchiveInterval(t *testing.T) {
	for _, gracePeriod := range []time.Duration{testShortGracePeriod, testIndexArchiveInterval} {
		config := testConfig(t.TempDir())
		config.ObjectStorage = newMemoryObjectStorage(t)
		config.IndexArchiveInterval = testIndexArchiveInterval
		config.LogicalDeleteGracePeriod = gracePeriod
		if err := CheckConfig(config); !Has(err, InvalidArgument) {
			t.Fatalf(testWantInvalidArgumentFormat, err)
		}
		if _, err := Open(config); !Has(err, InvalidArgument) {
			t.Fatalf(testWantInvalidArgumentFormat, err)
		}
		if _, err := RestoreForce(config); !Has(err, InvalidArgument) {
			t.Fatalf(testWantInvalidArgumentFormat, err)
		}
	}
	// without object storage there is no index backup, so the check does not apply
	config := testConfig(t.TempDir())
	config.IndexArchiveInterval = testIndexArchiveInterval
	config.LogicalDeleteGracePeriod = testShortGracePeriod
	if err := CheckConfig(config); err != nil {
		t.Fatal(err)
	}
	openTestDaemon(t, config)
}
