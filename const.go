package yaagwadb

const (
	NotFound        = "yaagwadb: not found"
	Corrupted       = "yaagwadb: corrupted" // failed entry, or the file data cannot be retrieved
	Exists          = "yaagwadb: already exists"
	NotDirectory    = "yaagwadb: not a directory"
	IsDirectory     = "yaagwadb: is a directory"
	NotEmpty        = "yaagwadb: directory not empty"
	NotSupported    = "yaagwadb: not supported"
	InvalidArgument = "yaagwadb: invalid argument"
	NoSpace         = "yaagwadb: no space left on device"
	Closed          = "yaagwadb: daemon closed"
	Locked          = "yaagwadb: data directory is locked by another process"
)

// errorMessages are error message constants carried over gRPC and matched by the receiver.
var errorMessages = []string{
	NotFound, Corrupted, Exists, NotDirectory, IsDirectory, NotEmpty, NotSupported,
	InvalidArgument, NoSpace, Closed, Locked,
}

const (
	errorFormatQuotedName  = "%w: %q"       // appends the quoted entry name
	errorFormatInodeNumber = "%w: inode %d" // appends the inode number
)

const (
	legacyIndexFileName         = "index.db"  // bbolt file of an older version
	fileDataDirectoryName       = "filedata"  // directory holding the file data
	temporaryFileDirectoryName  = "temporary" // directory for in-progress temporary files; fully cleared at startup
	dataDirectoryLockFileName   = "LOCK"      // flock target preventing multiple processes from opening the same data directory
	dataDirectoryPermission     = 0o700       // directories under the data directory are accessible only to the daemon user
	dataDirectoryFilePermission = 0o600       // files under the data directory are also accessible only to the daemon user
)

// sqliteJournalSuffixes are appended to the name of an SQLite file in WAL mode to name its journal files.
var sqliteJournalSuffixes = []string{"-wal", "-shm"}
