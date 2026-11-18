package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"yaagwadb"
	"yaagwadb/protocol"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

const commandImportDescription = "Import the files of a directory (a gRPC client of the daemon)"

const (
	flagSource               = "source"
	flagSourceShort          = "s"
	flagTo                   = "to"
	flagForceShort           = "f"
	flagSourceDescription    = "Directory to import (required)"
	flagToDescription        = "Directory in the database to import into. Missing directories are created. If omitted, the root"
	flagOverwriteDescription = "Overwrite files that already exist in the database. Without it, they are skipped"
	sourceDefault            = ""
	toDefault                = ""
	overwriteDefault         = false
)

const (
	ImportSourceRequired     = "yaagwadb: no import source specified"
	ImportSourceNotDirectory = "yaagwadb: import source is not a directory"
	ImportFailed             = "yaagwadb: import failed"
)

const (
	importSourceRequiredFormat     = "%w. Pass it with %s%s\nUsage: %s\nSee %s for details"
	importSourceNotDirectoryFormat = "%w: %s"
	importFailedFormat             = "%w: %d entries"
	importEntryErrorFormat         = "%s: %w"
	importExistingFormat           = "%w: %q already exists"
	importSummaryFormat            = "imported %d files, created %d directories, skipped %d existing files, ignored %d unsupported entries, failed %d\n"
	importIgnoredFormat            = "ignored (not a regular file or directory): %s\n"
	importFailureFormat            = "failed: %v\n"
)

const (
	// databasePathSeparator separates the names of a path in the database.
	databasePathSeparator = "/"
	// sourceRoot is the relative path of the source directory itself.
	sourceRoot = "."
	// noInode means no directory in the database corresponds to a source directory.
	noInode = 0
	// unknownOwner is the uid and gid used when the source file system does not report them.
	unknownOwner = 0
	// xattrNameSeparator separates the names in a listxattr(2) result.
	xattrNameSeparator = "\x00"
	// emptyXattrSize is the size of an empty listxattr(2) or getxattr(2) result.
	emptyXattrSize = 0
)

// skippedXattrNamespaces are Linux xattr namespaces that describe the source file system
// (security labels, ACLs, kernel-private data) rather than the file, so they are not copied.
var skippedXattrNamespaces = []string{"security.", "system.", "trusted."}

func newImportCommand(load configLoader) *cobra.Command {
	var source, to string
	var overwrite bool
	command := &cobra.Command{
		Use:   commandImport,
		Short: commandImportDescription,
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			if source == sourceDefault {
				return fmt.Errorf(importSourceRequiredFormat, errors.New(ImportSourceRequired), shortFlagPrefix, flagSourceShort, command.UseLine(), helpCommand(command))
			}
			configuration, err := load()
			if err != nil {
				return err
			}
			client, err := yaagwadb.Dial(configuration.Socket)
			if err != nil {
				return err
			}
			defer client.Close()
			if _, err := client.Status(command.Context()); err != nil {
				return fmt.Errorf(daemonUnreachableFormat, errors.New(DaemonUnreachable), configuration.Socket, err)
			}
			result, err := runImport(command.Context(), client, source, to, overwrite, command.ErrOrStderr())
			if err != nil {
				return err
			}
			result.report(command.OutOrStdout(), command.ErrOrStderr())
			if len(result.failures) > 0 {
				return fmt.Errorf(importFailedFormat, errors.New(ImportFailed), len(result.failures))
			}
			return nil
		},
	}
	command.Flags().StringVarP(&source, flagSource, flagSourceShort, sourceDefault, flagSourceDescription)
	command.Flags().StringVar(&to, flagTo, toDefault, flagToDescription)
	command.Flags().BoolVarP(&overwrite, flagForce, flagForceShort, overwriteDefault, flagOverwriteDescription)
	return command
}

// importer holds the state of one import: the counts and the failures of the entries.
type importer struct {
	ctx         context.Context
	client      *yaagwadb.Client
	overwrite   bool
	warnings    io.Writer
	files       int
	directories int
	skipped     int
	ignored     int
	failures    []error
}

// runImport walks source and writes its directories and regular files under to in the database.
// An entry that fails is recorded and the walk goes on; an error is returned only when nothing can be imported.
func runImport(ctx context.Context, client *yaagwadb.Client, source, to string, overwrite bool, warnings io.Writer) (*importer, error) {
	resolved, err := filepath.EvalSymlinks(source)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(importSourceNotDirectoryFormat, errors.New(ImportSourceNotDirectory), source)
	}
	r := &importer{ctx: ctx, client: client, overwrite: overwrite, warnings: warnings}
	root, err := r.targetDirectory(to, info)
	if err != nil {
		return nil, err
	}
	parents := map[string]uint64{sourceRoot: root}
	err = filepath.WalkDir(resolved, func(path string, entry fs.DirEntry, walkError error) error {
		relative, err := filepath.Rel(resolved, path)
		if err != nil {
			return err
		}
		if walkError != nil {
			r.fail(relative, walkError)
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if relative == sourceRoot {
			return nil
		}
		parent := parents[filepath.Dir(relative)]
		switch {
		case entry.IsDir():
			inode, err := r.directory(parent, entry, path)
			if err != nil {
				r.fail(relative, err)
			}
			if inode == noInode {
				return filepath.SkipDir
			}
			parents[relative] = inode
		case entry.Type().IsRegular():
			if err := r.file(parent, entry.Name(), path); err != nil {
				r.fail(relative, err)
			}
		default:
			r.ignored++
			fmt.Fprintf(r.warnings, importIgnoredFormat, relative)
		}
		return nil
	})
	return r, err
}

// targetDirectory finds or creates the directories of to, giving the created ones the attributes of the source directory.
func (r *importer) targetDirectory(to string, info fs.FileInfo) (uint64, error) {
	parent := uint64(yaagwadb.RootInodeNumber)
	for _, name := range strings.Split(strings.Trim(to, databasePathSeparator), databasePathSeparator) {
		if name == "" {
			continue
		}
		inode, _, err := r.findOrMakeDirectory(parent, name, info)
		if err != nil {
			return noInode, err
		}
		parent = inode
	}
	return parent, nil
}

// findOrMakeDirectory returns the directory name under parent, creating it if it does not exist.
func (r *importer) findOrMakeDirectory(parent uint64, name string, info fs.FileInfo) (uint64, bool, error) {
	existing, err := r.client.Lookup(r.ctx, parent, name)
	if err == nil {
		if !existing.IsDirectory() {
			return noInode, false, fmt.Errorf(importExistingFormat, errors.New(yaagwadb.NotDirectory), name)
		}
		return existing.InodeNumber, false, nil
	}
	if !yaagwadb.Has(err, yaagwadb.NotFound) {
		return noInode, false, err
	}
	mode, uid, gid := attributesOf(info)
	made, err := r.client.Mkdir(r.ctx, parent, name, mode, uid, gid)
	if err != nil {
		return noInode, false, err
	}
	r.directories++
	return made.InodeNumber, true, nil
}

// directory imports one source directory. It returns noInode when its entries cannot be imported.
func (r *importer) directory(parent uint64, entry fs.DirEntry, path string) (uint64, error) {
	info, err := entry.Info()
	if err != nil {
		return noInode, err
	}
	inode, created, err := r.findOrMakeDirectory(parent, entry.Name(), info)
	if err != nil || !created {
		return inode, err
	}
	return inode, r.copyXattrs(inode, path)
}

// file imports one regular file: its content, mode, owner, modification time and xattrs.
func (r *importer) file(parent uint64, name, path string) error {
	existing, err := r.client.Lookup(r.ctx, parent, name)
	switch {
	case err == nil && existing.IsDirectory():
		return fmt.Errorf(importExistingFormat, errors.New(yaagwadb.IsDirectory), name)
	case err == nil && !r.overwrite:
		r.skipped++
		return nil
	case err != nil && !yaagwadb.Has(err, yaagwadb.NotFound):
		return err
	}
	content, err := os.Open(path)
	if err != nil {
		return err
	}
	defer content.Close()
	info, err := content.Stat()
	if err != nil {
		return err
	}
	mode, uid, gid := attributesOf(info)
	response, err := r.client.Write(r.ctx, &protocol.WriteTarget{ParentInode: parent, Name: name, Mode: mode, Uid: uid, Gid: gid}, content)
	if err != nil {
		return err
	}
	inode := response.Attributes.GetInode()
	modifyTime := info.ModTime().UnixNano()
	if _, err := r.client.Setattr(r.ctx, &protocol.SetattrRequest{Inode: inode, ModifyTime: &modifyTime}); err != nil {
		return err
	}
	r.files++
	return r.copyXattrs(inode, path)
}

func (r *importer) copyXattrs(inode uint64, path string) error {
	names, err := listXattrs(path)
	if err != nil {
		return err
	}
	for _, name := range names {
		value, err := getXattr(path, name)
		if err != nil {
			return err
		}
		if err := r.client.Setxattr(r.ctx, inode, name, value); err != nil {
			return err
		}
	}
	return nil
}

func (r *importer) fail(relative string, err error) {
	r.failures = append(r.failures, fmt.Errorf(importEntryErrorFormat, relative, err))
}

func (r *importer) report(output, errorOutput io.Writer) {
	for _, failure := range r.failures {
		fmt.Fprintf(errorOutput, importFailureFormat, failure)
	}
	fmt.Fprintf(output, importSummaryFormat, r.files, r.directories, r.skipped, r.ignored, len(r.failures))
}

func attributesOf(info fs.FileInfo) (mode, uid, gid uint32) {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint32(stat.Mode), stat.Uid, stat.Gid
	}
	return uint32(info.Mode().Perm()), unknownOwner, unknownOwner
}

// listXattrs returns the names of the xattrs of path to copy. A file system without xattrs has none.
func listXattrs(path string) ([]string, error) {
	size, err := unix.Listxattr(path, nil)
	if errors.Is(err, unix.ENOTSUP) {
		return nil, nil
	}
	if err != nil || size == emptyXattrSize {
		return nil, err
	}
	buffer := make([]byte, size)
	if size, err = unix.Listxattr(path, buffer); err != nil {
		return nil, err
	}
	var names []string
	for _, name := range strings.Split(string(buffer[:size]), xattrNameSeparator) {
		if name != "" && !skippedXattrNamespace(name) {
			names = append(names, name)
		}
	}
	return names, nil
}

func skippedXattrNamespace(name string) bool {
	for _, namespace := range skippedXattrNamespaces {
		if strings.HasPrefix(name, namespace) {
			return true
		}
	}
	return false
}

func getXattr(path, name string) ([]byte, error) {
	size, err := unix.Getxattr(path, name, nil)
	if err != nil || size == emptyXattrSize {
		return []byte{}, err
	}
	buffer := make([]byte, size)
	if size, err = unix.Getxattr(path, name, buffer); err != nil {
		return nil, err
	}
	return buffer[:size], nil
}
