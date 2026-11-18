package yaagwadb

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"iter"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/pkg/errors"

	objectstorage "yaagwadb/object_storage"
)

const (
	// Message of the error that simulates an unreachable store
	testUnreachableMessage = "unreachable"
	// Prefix for object storage keys
	testObjectStoragePrefix = "tenant"
	// Separator between the key prefix and the rest
	testObjectKeySeparator = "/"
	// Length to read from the start of the index file (the SQLite header string)
	testIndexHeaderLength = 16
	// File data placed as already PUT
	testAlreadyPutFileData = "x"
	// String that is invalid as a fileDataID
	testInvalidFileDataID = "not-a-file-data-id"
	// Format reporting a missing object
	testObjectNotFoundFormat = "%w: %s"
	// Value when Head cannot return a size because the store is unreachable
	testUnknownSize = 0
	// Failure message format when the Head result differs
	testHeadFormat = "head %d %v"
	// Failure message format when the index archive is not an SQLite file
	testNotSQLiteFileFormat = "not an SQLite file: %q"
	// testSQLiteHeader starts every SQLite file
	testSQLiteHeader = "SQLite format 3\x00"
)

// failingObjectStorage simulates unreachable object storage. Once recovered is set, calls go to it.
type failingObjectStorage struct {
	recovered atomic.Pointer[memoryObjectStorage]
}

func (f *failingObjectStorage) Put(ctx context.Context, key string, reader io.Reader, size int64) error {
	if healthy := f.recovered.Load(); healthy != nil {
		return healthy.Put(ctx, key, reader, size)
	}
	return errors.New(testUnreachableMessage)
}
func (f *failingObjectStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if healthy := f.recovered.Load(); healthy != nil {
		return healthy.Get(ctx, key)
	}
	return nil, errors.New(testUnreachableMessage)
}
func (f *failingObjectStorage) Head(ctx context.Context, key string) (int64, error) {
	if healthy := f.recovered.Load(); healthy != nil {
		return healthy.Head(ctx, key)
	}
	return testUnknownSize, errors.New(testUnreachableMessage)
}
func (f *failingObjectStorage) Delete(ctx context.Context, key string) error {
	if healthy := f.recovered.Load(); healthy != nil {
		return healthy.Delete(ctx, key)
	}
	return errors.New(testUnreachableMessage)
}
func (f *failingObjectStorage) List(ctx context.Context, prefix string) iter.Seq2[string, error] {
	if healthy := f.recovered.Load(); healthy != nil {
		return healthy.List(ctx, prefix)
	}
	return func(yield func(string, error) bool) { yield(testEmptyString, errors.New(testUnreachableMessage)) }
}

func openWithFailingStorage(t *testing.T, config Config, storage *failingObjectStorage) *Daemon {
	t.Helper()
	first, err := Open(config)
	if err != nil {
		t.Fatal(err)
	}
	first.Close()
	config.ObjectStorage = storage
	return openTestDaemon(t, config)
}

// memoryObjectStorage is an in-memory object storage for tests.
type memoryObjectStorage struct {
	mutex   sync.Mutex
	objects map[string][]byte
}

func newMemoryObjectStorage(t *testing.T) *memoryObjectStorage {
	t.Helper()
	return &memoryObjectStorage{objects: map[string][]byte{}}
}

func (m *memoryObjectStorage) Put(_ context.Context, key string, reader io.Reader, _ int64) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.objects[key] = data
	return nil
}

func (m *memoryObjectStorage) object(key string) ([]byte, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	data, exists := m.objects[key]
	if !exists {
		return nil, fmt.Errorf(testObjectNotFoundFormat, errors.New(objectstorage.NotFound), key)
	}
	return data, nil
}

func (m *memoryObjectStorage) Get(_ context.Context, key string) (io.ReadCloser, error) {
	data, err := m.object(key)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (m *memoryObjectStorage) Head(_ context.Context, key string) (int64, error) {
	data, err := m.object(key)
	if err != nil {
		return testUnknownSize, err
	}
	return int64(len(data)), nil
}

func (m *memoryObjectStorage) Delete(_ context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.objects, key)
	return nil
}

func (m *memoryObjectStorage) List(_ context.Context, prefix string) iter.Seq2[string, error] {
	m.mutex.Lock()
	var keys []string
	for key := range m.objects {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	m.mutex.Unlock()
	slices.Sort(keys)
	return func(yield func(string, error) bool) {
		for _, key := range keys {
			if !yield(key, nil) {
				return
			}
		}
	}
}

func TestPutAndDeleteFileData(t *testing.T) {
	storage := newMemoryObjectStorage(t)
	config := testConfig(t.TempDir())
	config.ObjectStorage = storage
	config.ObjectStoragePrefix = testObjectStoragePrefix
	daemon := openTestDaemon(t, config)
	inode := writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFileData))

	key := testObjectStoragePrefix + testObjectKeySeparator + testObjectFileDataPrefix + inode.FileDataID
	size, err := storage.Head(context.Background(), key)
	if err != nil || size != int64(len(testFileData)) {
		t.Fatalf(testHeadFormat, size, err)
	}
	if err := daemon.deleteFileDataInObjectStorage(inode.FileDataID); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.Head(context.Background(), key); !objectstorage.Has(err, objectstorage.NotFound) {
		t.Fatalf(testWantNotFoundFormat, err)
	}
	// DELETE is idempotent
	if err := daemon.deleteFileDataInObjectStorage(inode.FileDataID); err != nil {
		t.Fatal(err)
	}
}

func TestPutIndexArchive(t *testing.T) {
	storage := newMemoryObjectStorage(t)
	config := testConfig(t.TempDir())
	config.ObjectStorage = storage
	daemon := openTestDaemon(t, config)
	writeAndWait(t, daemon, RootInodeNumber, testFileName, []byte(testFileData))
	if err := daemon.putIndexArchive(); err != nil {
		t.Fatal(err)
	}
	reader, err := storage.Get(context.Background(), objectIndexLatest)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	header := make([]byte, testIndexHeaderLength)
	if _, err := io.ReadFull(reader, header); err != nil {
		t.Fatal(err)
	}
	if string(header) != testSQLiteHeader {
		t.Fatalf(testNotSQLiteFileFormat, header)
	}
}

func TestPutFileDataOfMissingLocalFile(t *testing.T) {
	storage := newMemoryObjectStorage(t)
	config := testConfig(t.TempDir())
	config.ObjectStorage = storage
	daemon := openTestDaemon(t, config)
	fileDataID := newFileDataID()
	if err := daemon.update(func(transaction *sql.Tx) error {
		_, err := registerFileData(transaction, &fileDataRegistration{Parent: RootInodeNumber, Name: testFileName, FileDataID: fileDataID})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := daemon.uploadFileData(fileDataID); !Has(err, NotFound) {
		t.Fatalf(testWantNotFoundFormat, err)
	}
	if err := storage.Put(context.Background(), testObjectFileDataPrefix+fileDataID, bytes.NewReader([]byte(testAlreadyPutFileData)), int64(len(testAlreadyPutFileData))); err != nil {
		t.Fatal(err)
	}
	// Even if missing locally, it is done if already PUT
	if err := daemon.uploadFileData(fileDataID); err != nil {
		t.Fatal(err)
	}
	if err := daemon.uploadFileData(testInvalidFileDataID); !Has(err, InvalidArgument) {
		t.Fatalf(testWantInvalidArgumentFormat, err)
	}
}
