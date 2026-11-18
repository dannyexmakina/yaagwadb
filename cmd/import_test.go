package main

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"yaagwadb"
)

const (
	importTestSocketDirectoryPattern = "yaagwa"
	importTestSystemTemporary        = ""
	importTestSocketFileName         = "daemon.sock"
	importTestFirstName              = "a.txt"
	importTestFirstContent           = "alpha"
	importTestChangedContent         = "alpha changed"
	importTestDirectoryName          = "sub"
	importTestSecondName             = "b.txt"
	importTestSecondContent          = "beta"
	importTestLinkName               = "link"
	importTestTarget                 = "/dest/inner/"
	importTestTargetPath             = "dest/inner"
	importTestDirectoryPermission    = 0o755
	importTestFilePermission         = 0o644
	importTestModifyTimeSeconds      = 1_700_000_000
	importTestNanoseconds            = 0
	importTestReadFromStart          = 0
	importTestReadToEnd              = 0
	importTestCountsFormat           = "files %d directories %d skipped %d ignored %d failures %v"
	importTestContentFormat          = "%s: content %q"
	importTestModifyTimeFormat       = "modify time %d, want %d"
	importTestWantNotDirectoryFormat = "want ImportSourceNotDirectory, got %v"
)

func startImportTestDaemon(t *testing.T) *yaagwadb.Client {
	t.Helper()
	config := yaagwadb.DefaultConfig()
	config.DataDirectory = t.TempDir()
	config.DisableScheduler = true
	config.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	daemon, err := yaagwadb.Open(config)
	if err != nil {
		t.Fatal(err)
	}
	// A Unix socket path has a short length limit, so the socket goes in a short temporary directory
	directory, err := os.MkdirTemp(importTestSystemTemporary, importTestSocketDirectoryPattern)
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(directory, importTestSocketFileName)
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
		os.RemoveAll(directory)
	})
	return client
}

// makeImportSource creates a.txt, sub/b.txt and a symbolic link, and returns the directory.
func makeImportSource(t *testing.T) string {
	t.Helper()
	source := t.TempDir()
	first := filepath.Join(source, importTestFirstName)
	if err := os.WriteFile(first, []byte(importTestFirstContent), importTestFilePermission); err != nil {
		t.Fatal(err)
	}
	modifyTime := time.Unix(importTestModifyTimeSeconds, importTestNanoseconds)
	if err := os.Chtimes(first, modifyTime, modifyTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(source, importTestDirectoryName), importTestDirectoryPermission); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, importTestDirectoryName, importTestSecondName), []byte(importTestSecondContent), importTestFilePermission); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(importTestFirstName, filepath.Join(source, importTestLinkName)); err != nil {
		t.Fatal(err)
	}
	return source
}

func lookupImported(t *testing.T, client *yaagwadb.Client, path string) *yaagwadb.Inode {
	t.Helper()
	parent := uint64(yaagwadb.RootInodeNumber)
	var inode *yaagwadb.Inode
	for _, name := range strings.Split(path, databasePathSeparator) {
		var err error
		if inode, err = client.Lookup(context.Background(), parent, name); err != nil {
			t.Fatal(err)
		}
		parent = inode.InodeNumber
	}
	return inode
}

func assertImportedContent(t *testing.T, client *yaagwadb.Client, path, want string) {
	t.Helper()
	inode := lookupImported(t, client, path)
	got, err := client.Read(context.Background(), inode.InodeNumber, importTestReadFromStart, importTestReadToEnd)
	if err != nil || string(got) != want {
		t.Fatalf(importTestContentFormat, path, got)
	}
}

func assertCounts(t *testing.T, r *importer, files, directories, skipped, ignored int) {
	t.Helper()
	if r.files != files || r.directories != directories || r.skipped != skipped || r.ignored != ignored || len(r.failures) != 0 {
		t.Fatalf(importTestCountsFormat, r.files, r.directories, r.skipped, r.ignored, r.failures)
	}
}

func TestImport(t *testing.T) {
	client := startImportTestDaemon(t)
	source := makeImportSource(t)
	ctx := context.Background()

	// dest, dest/inner and sub are created; the link is ignored
	result, err := runImport(ctx, client, source, importTestTarget, overwriteDefault, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	assertCounts(t, result, 2, 3, 0, 1)
	firstPath := importTestTargetPath + databasePathSeparator + importTestFirstName
	assertImportedContent(t, client, firstPath, importTestFirstContent)
	assertImportedContent(t, client, importTestTargetPath+databasePathSeparator+importTestDirectoryName+databasePathSeparator+importTestSecondName, importTestSecondContent)
	wantModifyTime := time.Unix(importTestModifyTimeSeconds, importTestNanoseconds).UnixNano()
	if got := lookupImported(t, client, firstPath).ModifyTime; got != wantModifyTime {
		t.Fatalf(importTestModifyTimeFormat, got, wantModifyTime)
	}

	// Existing files are skipped unless overwrite is set
	if err := os.WriteFile(filepath.Join(source, importTestFirstName), []byte(importTestChangedContent), importTestFilePermission); err != nil {
		t.Fatal(err)
	}
	result, err = runImport(ctx, client, source, importTestTarget, overwriteDefault, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	assertCounts(t, result, 0, 0, 2, 1)
	assertImportedContent(t, client, firstPath, importTestFirstContent)

	result, err = runImport(ctx, client, source, importTestTarget, true, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	assertCounts(t, result, 2, 0, 0, 1)
	assertImportedContent(t, client, firstPath, importTestChangedContent)
}

func TestImportRefusesFileSource(t *testing.T) {
	client := startImportTestDaemon(t)
	file := filepath.Join(makeImportSource(t), importTestFirstName)
	if _, err := runImport(context.Background(), client, file, toDefault, overwriteDefault, io.Discard); !yaagwadb.Has(err, ImportSourceNotDirectory) {
		t.Fatalf(importTestWantNotDirectoryFormat, err)
	}
}
