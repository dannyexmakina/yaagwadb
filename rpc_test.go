package yaagwadb

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"

	"yaagwadb/protocol"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// Name prefix of the temporary directory holding the socket (kept short for the Unix socket path length limit)
	testSocketDirectoryPrefix = "yaagwa"
	// Socket file name
	testSocketFileName = "daemon.sock"
	// Repeated unit and count of the content, spanning streaming boundaries
	testStreamFileDataUnit  = "0123456789"
	testStreamFileDataCount = 300_000
	// Offset and length for reading the whole file (0 means to the end)
	testReadWholeOffset = 0
	testReadWholeLength = 0
	// Offset and length for a partial read (5 bytes from the start of the second repeated unit)
	testReadPartOffset = int64(len(testStreamFileDataUnit))
	testReadPartLength = 5
	// Expected content of the partial read
	testReadPartExpected = "01234"
	// Link count of a regular file
	testRegularFileLinkCount = 1
	// Device number of a regular file
	testRegularFileDevice = 0
	// File name after the move
	testMovedFileName = "moved"
	// Nonexistent file name
	testMissingFileName = "missing"
	// Name and content of the file written in the error tests
	testErrorFirstName     = "a"
	testErrorFirstFileData = "x"
	// Size passed to an unsupported size-changing Setattr
	testUnsupportedSize = 5
	// Invalid name pointing outside the directory
	testEscapingName = "../x"
	// Directory argument for creating a temporary directory in the OS default location
	testDefaultTemporaryDirectory = ""
	// Failure message format for an unexpected Write response
	testResponseFormat = "response %+v"
	// Failure message format for an unexpected full read
	testReadBytesFormat = "read %d bytes, %v"
	// Failure message format for an unexpected partial read
	testReadPartFormat = "part %q %v"
	// Failure message format for an unexpected Getattr result
	testAttributesFormat = "attributes %+v %v"
	// Failure message format for an unexpected Getxattr result
	testExtendedAttributeFormat = "xattr %q %v"
	// Failure message format for an unexpected Statfs result
	testStatfsFormat = "statfs %+v %v"
	// Failure message format when NotSupported was expected
	testWantNotSupportedFormat = "want NotSupported, got %v"
	// Failure message format for an unexpected gRPC status code
	testStatusCodeFormat = "%s: code %v, want %v"
	// Failure message format when the message is lost in the round trip
	testRoundTripLostFormat = "%s: round trip lost the message: %v"
	// Failure message format for an unexpected status code of an unknown error
	testUnknownErrorCodeFormat = "unknown error: %v"
	// Failure message format when the text of an error without a constant reaches the client
	testInternalMessageFormat = "internal error message: %q"
	// Failure message format when a cancelled call is not converted to Canceled
	testCancelledCodeFormat = "%v: want Canceled, got %v"
	// Failure message when converting nil does not yield nil
	testNilConversionMessage = "nil"
)

func startGRPC(t *testing.T, config Config) (*Daemon, *Client) {
	t.Helper()
	daemon := openTestDaemon(t, config)
	socketDirectory, err := os.MkdirTemp(testDefaultTemporaryDirectory, testSocketDirectoryPrefix)
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(socketDirectory, testSocketFileName)
	server, err := ListenGRPC(daemon, socket)
	if err != nil {
		t.Fatal(err)
	}
	client, err := Dial(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		client.Close()
		server.Stop()
		os.RemoveAll(socketDirectory)
	})
	return daemon, client
}

func TestGRPCRoundTrip(t *testing.T) {
	config := testConfig(t.TempDir())
	config.ObjectStorage = newMemoryObjectStorage(t)
	daemon, client := startGRPC(t, config)
	ctx := context.Background()

	directory, err := client.Mkdir(ctx, RootInodeNumber, testDirectoryName, testDirectoryMode, testOwnerID, testOwnerID)
	if err != nil {
		t.Fatal(err)
	}
	content := bytes.Repeat([]byte(testStreamFileDataUnit), testStreamFileDataCount)
	response, err := client.Write(ctx, &protocol.WriteTarget{ParentInode: directory.InodeNumber, Name: testFileName, Mode: testFileMode},
		bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	if response.Size != uint64(len(content)) || !validFileDataID(response.FileDataId) || response.Attributes.GetInode() == testZeroCount {
		t.Fatalf(testResponseFormat, response)
	}
	waitIdle(t, daemon)

	inode, err := client.Lookup(ctx, directory.InodeNumber, testFileName)
	if err != nil {
		t.Fatal(err)
	}
	data, err := client.Read(ctx, inode.InodeNumber, testReadWholeOffset, testReadWholeLength)
	if err != nil || !bytes.Equal(data, content) {
		t.Fatalf(testReadBytesFormat, len(data), err)
	}
	part, err := client.Read(ctx, inode.InodeNumber, testReadPartOffset, testReadPartLength)
	if err != nil || string(part) != testReadPartExpected {
		t.Fatalf(testReadPartFormat, part, err)
	}
	entries, err := client.Readdir(ctx, directory.InodeNumber)
	if err != nil || len(entries) != testExpectedSingle || entries[testFirstElementIndex].Name != testFileName {
		t.Fatalf(testEntriesFormat, entries, err)
	}

	attributes, err := client.Service().Getattr(ctx, &protocol.GetattrRequest{Inode: inode.InodeNumber})
	if err != nil || attributes.Nlink != testRegularFileLinkCount || attributes.Rdev != testRegularFileDevice || attributes.BlockSize != BlockSize {
		t.Fatalf(testAttributesFormat, attributes, err)
	}

	mode := uint32(testChangedFileMode)
	if _, err := client.Setattr(ctx, &protocol.SetattrRequest{Inode: inode.InodeNumber, Mode: &mode}); err != nil {
		t.Fatal(err)
	}
	if err := client.Setxattr(ctx, inode.InodeNumber, testExtendedAttributeName, []byte(testExtendedAttributeValue)); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	value, err := client.Getxattr(ctx, inode.InodeNumber, testExtendedAttributeName)
	if err != nil || string(value) != testExtendedAttributeValue {
		t.Fatalf(testExtendedAttributeFormat, value, err)
	}
	names, err := client.Listxattr(ctx, inode.InodeNumber)
	if err != nil || len(names) != testExpectedSingle {
		t.Fatalf(testExtendedAttributeNamesFormat, names, err)
	}
	if err := client.Removexattr(ctx, inode.InodeNumber, testExtendedAttributeName); err != nil {
		t.Fatal(err)
	}

	if err := client.Rename(ctx, directory.InodeNumber, testFileName, RootInodeNumber, testMovedFileName); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if err := client.Unlink(ctx, RootInodeNumber, testMovedFileName); err != nil {
		t.Fatal(err)
	}
	if err := client.Rmdir(ctx, RootInodeNumber, testDirectoryName); err != nil {
		t.Fatal(err)
	}
	waitIdle(t, daemon)
	if _, err := client.Lookup(ctx, RootInodeNumber, testMovedFileName); !Has(err, NotFound) {
		t.Fatalf(testWantNotFoundFormat, err)
	}

	statfs, err := client.Statfs(ctx)
	if err != nil || statfs.BlockSize == testZeroCount || statfs.NameLength != maximumNameLength {
		t.Fatalf(testStatfsFormat, statfs, err)
	}
	statusResponse, err := client.Status(ctx)
	if err != nil || len(statusResponse.Queues) != len(queueNames) {
		t.Fatalf(testStatusFormat, statusResponse, err)
	}
	tasks, err := client.ListTasks(ctx, &protocol.ListTasksRequest{Queue: ObjectStorageWriteQueue})
	if err != nil || len(tasks.Tasks) == testZeroCount || tasks.Tasks[testFirstElementIndex].Kind == testEmptyString {
		t.Fatalf(testTasksFormat, tasks, err)
	}
}

func TestGRPCErrors(t *testing.T) {
	config := testConfig(t.TempDir())
	config.ObjectStorageWriteRetry = RetryPolicy{Base: testLongRetryWait, Cap: testLongRetryWait,
		MaxAttempts: testRetryMaxAttempts, MaxDuration: testLongRetryWait}
	// Create the index, then reconnect to an unreachable object storage (PUTs stay queued)
	initialized, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	initialized.Close()
	config.ObjectStorage = &failingObjectStorage{}
	_, client := startGRPC(t, config)
	ctx := context.Background()

	if _, err := client.Lookup(ctx, RootInodeNumber, testMissingFileName); !Has(err, NotFound) {
		t.Fatalf(testWantNotFoundFormat, err)
	}
	if _, err := client.Write(ctx, &protocol.WriteTarget{ParentInode: RootInodeNumber, Name: testErrorFirstName}, bytes.NewReader([]byte(testErrorFirstFileData))); err != nil {
		t.Fatal(err)
	}
	size := uint64(testUnsupportedSize)
	if _, err := client.Setattr(ctx, &protocol.SetattrRequest{Inode: RootInodeNumber, Size: &size}); !Has(err, NotSupported) {
		t.Fatalf(testWantNotSupportedFormat, err)
	}
	if _, err := client.Write(ctx, &protocol.WriteTarget{ParentInode: RootInodeNumber, Name: testEscapingName}, bytes.NewReader(nil)); !Has(err, InvalidArgument) {
		t.Fatalf(testWantInvalidArgumentFormat, err)
	}
}

func TestStatusConversion(t *testing.T) {
	for message, code := range statusCodes {
		converted := toStatus(fmt.Errorf(testErrorDetailFormat, errors.New(message)))
		if got := status.Code(converted); got != code {
			t.Errorf(testStatusCodeFormat, message, got, code)
		}
		if back := fromStatus(converted); !Has(back, message) {
			t.Errorf(testRoundTripLostFormat, message, back)
		}
	}
	internal := toStatus(errors.New(testOtherErrorMessage))
	if got := status.Code(internal); got != codes.Internal {
		t.Errorf(testUnknownErrorCodeFormat, got)
	}
	// The text of an error without a constant stays in the daemon log
	if got := status.Convert(internal).Message(); got != rpcInternalErrorMessage {
		t.Errorf(testInternalMessageFormat, got)
	}
	for _, err := range []error{context.Canceled, status.Error(codes.Canceled, testOtherErrorMessage)} {
		if got := status.Code(toStatus(err)); got != codes.Canceled {
			t.Errorf(testCancelledCodeFormat, err, got)
		}
	}
	if toStatus(nil) != nil || fromStatus(nil) != nil {
		t.Error(testNilConversionMessage)
	}
}
