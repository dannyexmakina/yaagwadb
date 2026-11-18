package mount

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/pkg/errors"

	"yaagwadb"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// mountTestSocketDirectoryPattern is the name pattern of the temporary directory for the Unix socket. Kept short to fit the path length limit.
	mountTestSocketDirectoryPattern = "yaagwa"
	// mountTestSystemTemporaryDirectory makes os.MkdirTemp use the OS default temporary directory.
	mountTestSystemTemporaryDirectory = ""
	// mountTestDisableScheduler stops the scheduler so periodic jobs do not interfere with the test.
	mountTestDisableScheduler = true
	// mountTestSocketFileName is the file name of the daemon's gRPC socket.
	mountTestSocketFileName = "daemon.sock"
	// mountTestLinuxOperatingSystem is the OS on which the FUSE integration tests run.
	mountTestLinuxOperatingSystem = "linux"
	// mountTestIntegrationEnvironmentVariable is the environment variable that enables the FUSE integration tests.
	mountTestIntegrationEnvironmentVariable = "YAAGWADB_FUSE_INTEGRATION"
	// mountTestIntegrationEnabledValue is the environment variable value that enables the integration tests.
	mountTestIntegrationEnabledValue = "1"
	// mountTestSkipMessage is the reason given when skipping the integration tests.
	mountTestSkipMessage = "FUSE integration tests run on Linux with YAAGWADB_FUSE_INTEGRATION=1"
	// mountTestEventuallyTimeout is the maximum wait for the asynchronous index commit.
	mountTestEventuallyTimeout = 5 * time.Second
	// mountTestEventuallyInterval is the interval between condition rechecks.
	mountTestEventuallyInterval = 10 * time.Millisecond
	// mountTestDirectoryName is the name of the directory created by the test.
	mountTestDirectoryName = "directory"
	// mountTestDirectoryPermission is the permission of the directory created by the test.
	mountTestDirectoryPermission = 0o755
	// mountTestFilePermission is the permission of the files created by the test.
	mountTestFilePermission = 0o644
	// mountTestFileDataUnit is the repeated unit that makes up the content.
	mountTestFileDataUnit = "0123456789"
	// mountTestFileDataRepeatCount is the repeat count of the content, large enough to span multiple chunks.
	mountTestFileDataRepeatCount = 100_000
	// mountTestFileName is the name of the file written by the test.
	mountTestFileName = "file"
	// mountTestRenamedFileName is the rename destination file name.
	mountTestRenamedFileName = "renamed"
	// mountTestHardLinkName is the name used when attempting to create a hard link.
	mountTestHardLinkName = "hard"
	// mountTestSymbolicLinkName is the name used when attempting to create a symbolic link.
	mountTestSymbolicLinkName = "soft"
	// mountTestPendingFileName is the file name written while PUTs to object storage are backed up.
	mountTestPendingFileName = "pending"
	// mountTestSecondFileName is the second file name written while PUTs to object storage are backed up.
	mountTestSecondFileName = "second"
	// mountTestFirstPayload and mountTestSecondPayload are the contents of small files.
	mountTestFirstPayload  = "x"
	mountTestSecondPayload = "y"
	// mountTestExtendedAttributeName and mountTestExtendedAttributeValue are the extended attribute to set.
	mountTestExtendedAttributeName  = "user.case"
	mountTestExtendedAttributeValue = "A-1"
	// mountTestExtendedAttributeBufferSize is the buffer size for reading extended attributes.
	mountTestExtendedAttributeBufferSize = 16
	// mountTestNoExtendedAttributeFlags means no flags are passed to setxattr(2).
	mountTestNoExtendedAttributeFlags = 0
	// mountTestLongRetryWait is the wait before retrying a failed PUT (long enough not to elapse during the test).
	mountTestLongRetryWait = time.Hour
	// mountTestRetryMaxAttempts is the maximum number of PUT retries.
	mountTestRetryMaxAttempts = 3
	// mountTestStorageUnreachableMessage is the error returned by the unreachable object storage.
	mountTestStorageUnreachableMessage = "object storage unreachable"
	// mountTestUnknownSize is the value Head returns when it cannot report a size.
	mountTestUnknownSize = -1
	// mountTestEmptyKey is the value List returns when it cannot return a key.
	mountTestEmptyKey = ""
	// mountTestReadBackFormat is the format used when a file cannot be read right after writing.
	mountTestReadBackFormat = "read back %q: %v"
	// mountTestUnreachableMessage is the message of an error without a constant.
	mountTestUnreachableMessage = "daemon unreachable"
	// mountTestErrorDetailFormat is the format for adding details to an error.
	mountTestErrorDetailFormat = "%w: detail"
	// mountTestOverwriteWriters and mountTestOverwriteRounds are the writers overwriting one file at once and how many times each overwrites it.
	mountTestOverwriteWriters = 4
	mountTestOverwriteRounds  = 20
	// mountTestOverwriteFileDataFormat makes the contents of the overwrites. All contents have the same length
	mountTestOverwriteFileDataFormat = "writer %02d round %02d"
	// mountTestTruncatedPayload is written after truncating through the open file.
	mountTestTruncatedPayload = "after truncate"
	// mountTestReadPieceLength is the size of each read in TestFUSEReadAtOffsets; mountTestReadTailLength is how far before the end a read that runs past the end starts.
	mountTestReadPieceLength = 100_000
	mountTestReadTailLength  = 10
	// Failure messages.
	mountTestReadAtFormat              = "read at %d: %v"
	mountTestReadAtMismatchFormat      = "read at %d: %d bytes do not match the %d bytes written"
	mountTestOverwriteErrorFormat      = "overwrite: %v"
	mountTestEmptySeenFormat           = "the file was seen empty %d times during overwrites"
	mountTestUnexpectedFinalFormat     = "final content %q is not one of the written contents"
	mountTestContentMismatch           = "content mismatch"
	mountTestExtendedAttributeMismatch = "extended attribute mismatch"
	mountTestFileShouldBeRemoved       = "file should be removed"
	mountTestLinkWantFormat            = "link: want EPERM, got %v"
	mountTestSymlinkWantFormat         = "symlink: want EPERM, got %v"
	mountTestErrnoMismatchFormat       = "%s: want %v, got %v"
)

func testConfig(directory string) yaagwadb.Config {
	return yaagwadb.Config{
		DataDirectory: directory, DisableScheduler: mountTestDisableScheduler,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func startDaemon(t *testing.T, config yaagwadb.Config) *yaagwadb.Client {
	t.Helper()
	socketDirectory, err := os.MkdirTemp(mountTestSystemTemporaryDirectory, mountTestSocketDirectoryPattern)
	if err != nil {
		t.Fatal(err)
	}
	daemon, err := yaagwadb.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(socketDirectory, mountTestSocketFileName)
	server, err := yaagwadb.ListenGRPC(daemon, socket)
	if err != nil {
		t.Fatal(err)
	}
	client, err := yaagwadb.Dial(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		client.Close()
		server.Stop()
		daemon.Close()
		os.RemoveAll(socketDirectory)
	})
	return client
}

func requireFUSE(t *testing.T) {
	t.Helper()
	if runtime.GOOS != mountTestLinuxOperatingSystem || os.Getenv(mountTestIntegrationEnvironmentVariable) != mountTestIntegrationEnabledValue {
		t.Skip(mountTestSkipMessage)
	}
}

func mountForTest(t *testing.T, config yaagwadb.Config) string {
	t.Helper()
	requireFUSE(t)
	client := startDaemon(t, config)
	mountpoint := t.TempDir()
	server, err := Start(client, mountpoint, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.WaitMount(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { server.Unmount() })
	return mountpoint
}

func eventually(t *testing.T, check func() error) {
	t.Helper()
	deadline := time.Now().Add(mountTestEventuallyTimeout)
	for {
		err := check()
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(mountTestEventuallyInterval)
	}
}

func TestFUSEOperations(t *testing.T) {
	mountpoint := mountForTest(t, testConfig(t.TempDir()))
	directory := filepath.Join(mountpoint, mountTestDirectoryName)
	if err := os.Mkdir(directory, mountTestDirectoryPermission); err != nil {
		t.Fatal(err)
	}

	content := bytes.Repeat([]byte(mountTestFileDataUnit), mountTestFileDataRepeatCount)
	path := filepath.Join(directory, mountTestFileName)
	if err := os.WriteFile(path, content, mountTestFilePermission); err != nil {
		t.Fatal(err)
	}
	// EAGAIN until committed to the index; readable after the commit
	eventually(t, func() error {
		got, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(got, content) {
			return errors.New(mountTestContentMismatch)
		}
		return nil
	})

	if err := unix.Setxattr(path, mountTestExtendedAttributeName, []byte(mountTestExtendedAttributeValue), mountTestNoExtendedAttributeFlags); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() error {
		buffer := make([]byte, mountTestExtendedAttributeBufferSize)
		n, err := unix.Getxattr(path, mountTestExtendedAttributeName, buffer)
		if err != nil {
			return err
		}
		if string(buffer[:n]) != mountTestExtendedAttributeValue {
			return errors.New(mountTestExtendedAttributeMismatch)
		}
		return nil
	})

	renamed := filepath.Join(mountpoint, mountTestRenamedFileName)
	if err := os.Rename(path, renamed); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() error { _, err := os.Stat(renamed); return err })

	if err := os.Remove(renamed); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() error {
		if _, err := os.Stat(renamed); !errors.Is(err, os.ErrNotExist) {
			return errors.New(mountTestFileShouldBeRemoved)
		}
		return nil
	})
	if err := os.Remove(directory); err != nil {
		t.Fatal(err)
	}
	var statfs unix.Statfs_t
	if err := unix.Statfs(mountpoint, &statfs); err != nil {
		t.Fatal(err)
	}
}

func TestFUSERejectsLinks(t *testing.T) {
	mountpoint := mountForTest(t, testConfig(t.TempDir()))
	path := filepath.Join(mountpoint, mountTestFileName)
	if err := os.WriteFile(path, []byte(mountTestFirstPayload), mountTestFilePermission); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() error { _, err := os.Stat(path); return err })
	if err := os.Link(path, filepath.Join(mountpoint, mountTestHardLinkName)); !errors.Is(err, syscall.EPERM) {
		t.Fatalf(mountTestLinkWantFormat, err)
	}
	if err := os.Symlink(path, filepath.Join(mountpoint, mountTestSymbolicLinkName)); !errors.Is(err, syscall.EPERM) {
		t.Fatalf(mountTestSymlinkWantFormat, err)
	}
}

// Overwriting one file from several writers at once must not fail, and readers must never see it empty.
func TestFUSEConcurrentOverwrite(t *testing.T) {
	mountpoint := mountForTest(t, testConfig(t.TempDir()))
	path := filepath.Join(mountpoint, mountTestFileName)
	contents := map[string]bool{}
	content := func(writer, round int) []byte {
		return []byte(fmt.Sprintf(mountTestOverwriteFileDataFormat, writer, round))
	}
	for writer := range mountTestOverwriteWriters {
		for round := range mountTestOverwriteRounds {
			contents[string(content(writer, round))] = true
		}
	}
	if err := os.WriteFile(path, content(0, 0), mountTestFilePermission); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() error { _, err := os.ReadFile(path); return err })

	var group sync.WaitGroup
	var mutex sync.Mutex
	var failures []error
	done := make(chan struct{})
	for writer := range mountTestOverwriteWriters {
		group.Go(func() {
			for round := range mountTestOverwriteRounds {
				if err := os.WriteFile(path, content(writer, round), mountTestFilePermission); err != nil {
					mutex.Lock()
					failures = append(failures, err)
					mutex.Unlock()
				}
			}
		})
	}
	emptySeen := 0
	var watcher sync.WaitGroup
	watcher.Go(func() {
		for {
			select {
			case <-done:
				return
			default:
			}
			if info, err := os.Stat(path); err == nil && info.Size() == 0 {
				emptySeen++
			}
		}
	})
	group.Wait()
	close(done)
	watcher.Wait()

	for _, err := range failures {
		t.Errorf(mountTestOverwriteErrorFormat, err)
	}
	if emptySeen > 0 {
		t.Errorf(mountTestEmptySeenFormat, emptySeen)
	}
	eventually(t, func() error {
		got, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !contents[string(got)] {
			return fmt.Errorf(mountTestUnexpectedFinalFormat, got)
		}
		return nil
	})
}

func TestFUSEReadAtOffsets(t *testing.T) {
	mountpoint := mountForTest(t, testConfig(t.TempDir()))
	content := bytes.Repeat([]byte(mountTestFileDataUnit), mountTestFileDataRepeatCount)
	path := filepath.Join(mountpoint, mountTestFileName)
	if err := os.WriteFile(path, content, mountTestFilePermission); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() error { _, err := os.Stat(path); return err })
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	length := len(content)
	// Sequential, then backwards, then past the end
	for _, offset := range []int{0, mountTestReadPieceLength, 2 * mountTestReadPieceLength, length / 2, mountTestReadPieceLength, length - mountTestReadTailLength, length} {
		buffer := make([]byte, mountTestReadPieceLength)
		n, err := file.ReadAt(buffer, int64(offset))
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf(mountTestReadAtFormat, offset, err)
		}
		want := content[offset:min(offset+mountTestReadPieceLength, length)]
		if !bytes.Equal(buffer[:n], want) {
			t.Fatalf(mountTestReadAtMismatchFormat, offset, n, len(want))
		}
	}
}

// ftruncate(2) on a file opened for writing without O_TRUNC truncates that write.
func TestFUSETruncateThroughOpenFile(t *testing.T) {
	mountpoint := mountForTest(t, testConfig(t.TempDir()))
	path := filepath.Join(mountpoint, mountTestFileName)
	if err := os.WriteFile(path, []byte(mountTestFileDataUnit), mountTestFilePermission); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() error { _, err := os.ReadFile(path); return err })
	file, err := os.OpenFile(path, os.O_WRONLY, mountTestFilePermission)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(0); err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(mountTestTruncatedPayload); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() error {
		got, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(got) != mountTestTruncatedPayload {
			return errors.New(mountTestContentMismatch)
		}
		return nil
	})
}

// Writes are accepted while PUTs to object storage are pending (no EBUSY).
func TestFUSEWriteWhilePutsPending(t *testing.T) {
	config := testConfig(t.TempDir())
	// Even when a PUT fails, the task stays queued while waiting for the retry
	config.ObjectStorageWriteRetry = yaagwadb.RetryPolicy{Base: mountTestLongRetryWait, Cap: mountTestLongRetryWait,
		MaxAttempts: mountTestRetryMaxAttempts, MaxDuration: mountTestLongRetryWait}
	// Create the index, then reconnect to an unreachable object storage
	initialized, err := yaagwadb.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	initialized.Close()
	config.ObjectStorage = unreachableObjectStorage{}
	mountpoint := mountForTest(t, config)

	path := filepath.Join(mountpoint, mountTestPendingFileName)
	if err := os.WriteFile(path, []byte(mountTestFirstPayload), mountTestFilePermission); err != nil {
		t.Fatal(err)
	}
	// A write is in the index once it responds, so it is readable immediately
	if data, err := os.ReadFile(path); err != nil || string(data) != mountTestFirstPayload {
		t.Fatalf(mountTestReadBackFormat, data, err)
	}
	second := filepath.Join(mountpoint, mountTestSecondFileName)
	if err := os.WriteFile(second, []byte(mountTestSecondPayload), mountTestFilePermission); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(second); err != nil || string(data) != mountTestSecondPayload {
		t.Fatalf(mountTestReadBackFormat, data, err)
	}
}

type unreachableObjectStorage struct{}

func (unreachableObjectStorage) Put(context.Context, string, io.Reader, int64) error {
	return errors.New(mountTestStorageUnreachableMessage)
}

func (unreachableObjectStorage) Get(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New(mountTestStorageUnreachableMessage)
}

func (unreachableObjectStorage) Head(context.Context, string) (int64, error) {
	return mountTestUnknownSize, errors.New(mountTestStorageUnreachableMessage)
}

func (unreachableObjectStorage) Delete(context.Context, string) error {
	return errors.New(mountTestStorageUnreachableMessage)
}

func (unreachableObjectStorage) List(context.Context, string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		yield(mountTestEmptyKey, errors.New(mountTestStorageUnreachableMessage))
	}
}

func TestErrnoOf(t *testing.T) {
	for message, want := range map[string]syscall.Errno{
		yaagwadb.NotFound: syscall.ENOENT, yaagwadb.Corrupted: syscall.EIO,
		yaagwadb.NoSpace: syscall.ENOSPC, yaagwadb.Exists: syscall.EEXIST,
		yaagwadb.NotDirectory: syscall.ENOTDIR, yaagwadb.IsDirectory: syscall.EISDIR, yaagwadb.NotEmpty: syscall.ENOTEMPTY,
		yaagwadb.NotSupported: syscall.ENOTSUP, yaagwadb.InvalidArgument: syscall.EINVAL, yaagwadb.Closed: syscall.EIO,
		mountTestUnreachableMessage: syscall.EIO,
	} {
		if got := errnoOf(fmt.Errorf(mountTestErrorDetailFormat, errors.New(message))); got != want {
			t.Errorf(mountTestErrnoMismatchFormat, message, want, got)
		}
	}
	for _, err := range []error{context.Canceled, status.Error(codes.Canceled, context.Canceled.Error())} {
		if got := errnoOf(fmt.Errorf(mountTestErrorDetailFormat, err)); got != syscall.EINTR {
			t.Errorf(mountTestErrnoMismatchFormat, err, syscall.EINTR, got)
		}
	}
}
