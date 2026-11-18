package mount

import (
	"context"
	"io"
	"os"
	"sync"
	"syscall"
	"time"

	"yaagwadb"
	"yaagwadb/protocol"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// fileSystemName is the file system and device name shown in the mount list.
	fileSystemName = "yaagwadb"
	// noKernelCacheTimeout is the timeout that keeps the kernel from caching entries and attributes. The mounter has no cache.
	noKernelCacheTimeout = time.Duration(0)
	// statBlockUnit is the unit counted by stat's st_blocks (fixed at 512 bytes by POSIX).
	statBlockUnit = 512
	// statBlockRoundUp is added to Size to round it up to a multiple of statBlockUnit.
	statBlockRoundUp = statBlockUnit - 1
	// regularFileLinkCount is the link count of a regular file (hard links are not supported).
	regularFileLinkCount = 1
	// directoryLinkCount is the link count of a directory (its own name and ".").
	directoryLinkCount = 2
	// unregisteredInodeNumber is the inode number of a node being written and not yet registered in the index.
	unregisteredInodeNumber = 0
	// noOpenFlags means no flags are passed to FUSE in Open and Create responses.
	noOpenFlags = 0
	// noRenameFlags means rename(2) has no flags such as RENAME_NOREPLACE.
	noRenameFlags = 0
	// unknownCallerIdentifier is the uid and gid (root) used when the caller is unknown.
	unknownCallerIdentifier = 0
	// extendedAttributeNameTerminator is the NUL separating names in a listxattr(2) response.
	extendedAttributeNameTerminator = 0
	// noMatchingFlagBits means the AND of the flags and the mask has no bits set.
	noMatchingFlagBits = 0
	// emptyLength is the length for size queries and empty responses.
	emptyLength = 0
)

// WriteCancelled means the write was cancelled because the file was closed without writing anything.
const WriteCancelled = "mount: write cancelled"

// Options is the FUSE mount configuration.
type Options struct {
	Debug bool
	// AllowOther lets users other than the mounting user access the mount. The kernel then checks the mode bits (default_permissions)
	AllowOther bool
}

func Start(client *yaagwadb.Client, mountpoint string, options Options) (*fuse.Server, error) {
	noCache := noKernelCacheTimeout
	root := &node{client: client, inodeNumber: yaagwadb.RootInodeNumber}
	return fs.Mount(mountpoint, root, &fs.Options{
		// The mounter has no cache, and the kernel is not allowed one either
		EntryTimeout:    &noCache,
		AttrTimeout:     &noCache,
		NegativeTimeout: &noCache,
		// By default the root inode number is 0, which prevents Finder from identifying files
		RootStableAttr: &fs.StableAttr{Ino: yaagwadb.RootInodeNumber, Mode: yaagwadb.ModeDirectory},
		MountOptions: fuse.MountOptions{
			FsName: fileSystemName,
			Name:   fileSystemName,
			Debug:  options.Debug,
			// The mounter does no permission checks of its own, so other users are checked by the kernel
			AllowOther: options.AllowOther,
			Options:    defaultPermissionsOptions(options.AllowOther),
		},
	})
}

// defaultPermissionsOption is the mount option that makes the kernel check the mode bits.
const defaultPermissionsOption = "default_permissions"

func defaultPermissionsOptions(allowOther bool) []string {
	if !allowOther {
		return nil
	}
	return []string{defaultPermissionsOption}
}

func errnoOf(err error) syscall.Errno {
	if err == nil {
		return fs.OK
	}
	switch {
	// The kernel interrupts a request when the caller gets a signal (such as the Go runtime's preemption),
	// and go-fuse then cancels the call. EINTR lets the caller retry it
	case errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled:
		return syscall.EINTR
	case yaagwadb.Has(err, yaagwadb.NotFound):
		return syscall.ENOENT
	case yaagwadb.Has(err, yaagwadb.Corrupted):
		return syscall.EIO
	case yaagwadb.Has(err, yaagwadb.NoSpace):
		return syscall.ENOSPC
	case yaagwadb.Has(err, yaagwadb.Exists):
		return syscall.EEXIST
	case yaagwadb.Has(err, yaagwadb.NotDirectory):
		return syscall.ENOTDIR
	case yaagwadb.Has(err, yaagwadb.IsDirectory):
		return syscall.EISDIR
	case yaagwadb.Has(err, yaagwadb.NotEmpty):
		return syscall.ENOTEMPTY
	case yaagwadb.Has(err, yaagwadb.NotSupported):
		return syscall.ENOTSUP
	case yaagwadb.Has(err, yaagwadb.InvalidArgument):
		return syscall.EINVAL
	}
	return syscall.EIO
}

// node corresponds to one daemon inode. An inodeNumber of unregisteredInodeNumber means
// a file being written and not yet registered in the index.
type node struct {
	fs.Inode
	client      *yaagwadb.Client
	inodeNumber uint64
	session     *writeSession // the write when inodeNumber is unregisteredInodeNumber

	mutex sync.Mutex
	// truncators are writes opened without O_TRUNC, waiting for a size-0 SETATTR, by the thread ID of the caller.
	// The kernel sends that SETATTR from the thread that called open(2), so the thread ID pairs them
	// even when several writes to the file are opened at once
	truncators map[uint32]*writeSession
}

var (
	_ fs.NodeLookuper      = (*node)(nil)
	_ fs.NodeGetattrer     = (*node)(nil)
	_ fs.NodeSetattrer     = (*node)(nil)
	_ fs.NodeReaddirer     = (*node)(nil)
	_ fs.NodeOpener        = (*node)(nil)
	_ fs.NodeReader        = (*node)(nil)
	_ fs.NodeCreater       = (*node)(nil)
	_ fs.NodeMkdirer       = (*node)(nil)
	_ fs.NodeUnlinker      = (*node)(nil)
	_ fs.NodeRmdirer       = (*node)(nil)
	_ fs.NodeRenamer       = (*node)(nil)
	_ fs.NodeLinker        = (*node)(nil)
	_ fs.NodeSymlinker     = (*node)(nil)
	_ fs.NodeGetxattrer    = (*node)(nil)
	_ fs.NodeSetxattrer    = (*node)(nil)
	_ fs.NodeListxattrer   = (*node)(nil)
	_ fs.NodeRemovexattrer = (*node)(nil)
	_ fs.NodeStatfser      = (*node)(nil)
)

func fillAttribute(inode *yaagwadb.Inode, out *fuse.Attr) {
	out.Ino = inode.InodeNumber
	out.Size = inode.Size
	out.Blocks = (inode.Size + statBlockRoundUp) / statBlockUnit
	out.Mode = inode.Mode
	out.Nlink = regularFileLinkCount
	if inode.IsDirectory() {
		out.Nlink = directoryLinkCount
	}
	out.Owner = fuse.Owner{Uid: inode.UID, Gid: inode.GID}
	if inode.InodeNumber == yaagwadb.RootInodeNumber {
		// Show the root as owned by the mounting user, without changing the owner stored in the index
		out.Owner = fuse.Owner{Uid: uint32(os.Getuid()), Gid: uint32(os.Getgid())}
	}
	out.Blksize = yaagwadb.BlockSize
	setTime := func(nanoseconds int64, seconds *uint64, remainder *uint32) {
		*seconds = uint64(nanoseconds / int64(time.Second))
		*remainder = uint32(nanoseconds % int64(time.Second))
	}
	setTime(inode.AccessTime, &out.Atime, &out.Atimensec)
	setTime(inode.ModifyTime, &out.Mtime, &out.Mtimensec)
	setTime(inode.ChangeTime, &out.Ctime, &out.Ctimensec)
}

func (n *node) number() uint64 {
	if n.inodeNumber == unregisteredInodeNumber && n.session != nil {
		return n.session.registeredInodeNumber() // once the write is in the index, treat it as that inode
	}
	return n.inodeNumber
}

func (n *node) child(ctx context.Context, inode *yaagwadb.Inode) *fs.Inode {
	return n.NewInode(ctx, &node{client: n.client, inodeNumber: inode.InodeNumber},
		fs.StableAttr{Mode: inode.Mode & yaagwadb.ModeTypeMask, Ino: inode.InodeNumber})
}

func caller(ctx context.Context) (uid, gid uint32) {
	if caller, ok := fuse.FromContext(ctx); ok {
		return caller.Uid, caller.Gid
	}
	return unknownCallerIdentifier, unknownCallerIdentifier
}

// callerThread returns the thread ID of the caller, or unknownCallerIdentifier if it is unknown.
func callerThread(ctx context.Context) uint32 {
	if caller, ok := fuse.FromContext(ctx); ok {
		return caller.Pid
	}
	return unknownCallerIdentifier
}

// addTruncator records a write waiting for a size-0 SETATTR from the calling thread.
func (n *node) addTruncator(ctx context.Context, session *writeSession) {
	thread := callerThread(ctx)
	n.mutex.Lock()
	if n.truncators == nil {
		n.truncators = map[uint32]*writeSession{}
	}
	n.truncators[thread] = session
	n.mutex.Unlock()
	session.onClose = func() { n.removeTruncator(session) }
}

// takeTruncator removes and returns the write waiting for a size-0 SETATTR from the calling thread, or nil.
func (n *node) takeTruncator(ctx context.Context) *writeSession {
	thread := callerThread(ctx)
	n.mutex.Lock()
	defer n.mutex.Unlock()
	session := n.truncators[thread]
	delete(n.truncators, thread)
	return session
}

// removeTruncator forgets the write, so that a later SETATTR from the same thread is not passed to it.
func (n *node) removeTruncator(session *writeSession) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	for thread, waiting := range n.truncators {
		if waiting == session {
			delete(n.truncators, thread)
		}
	}
}

func (n *node) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	inode, err := n.client.Lookup(ctx, n.number(), name)
	if err != nil {
		return nil, errnoOf(err)
	}
	fillAttribute(inode, &out.Attr)
	return n.child(ctx, inode), fs.OK
}

func (n *node) Getattr(ctx context.Context, handle fs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	if n.number() == unregisteredInodeNumber {
		if n.session == nil {
			return syscall.EAGAIN
		}
		n.session.fillAttribute(&out.Attr)
		return fs.OK
	}
	inode, err := n.client.Getattr(ctx, n.number())
	if err != nil {
		return errnoOf(err)
	}
	fillAttribute(inode, &out.Attr)
	return fs.OK
}

func (n *node) Setattr(ctx context.Context, handle fs.FileHandle, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	if n.number() == unregisteredInodeNumber {
		if n.session == nil {
			return syscall.EAGAIN
		}
		errno := n.session.setAttribute(ctx, in)
		n.session.fillAttribute(&out.Attr)
		return errno
	}
	if session, ok := handle.(*writeSession); ok {
		// ftruncate(2) and the like on a file opened for writing apply to that write
		n.removeTruncator(session)
		return session.Setattr(ctx, in, out)
	}
	if size, ok := in.GetSize(); ok && size == emptyLength {
		// For O_TRUNC, the kernel sends this without a file handle from the thread that called open(2),
		// so pass it to the write that thread opened. A size-0 SETATTR from another thread is truncate(2) and goes to the daemon
		if truncator := n.takeTruncator(ctx); truncator != nil {
			return truncator.Setattr(ctx, in, out)
		}
	}
	request := &protocol.SetattrRequest{Inode: n.number()}
	if mode, ok := in.GetMode(); ok {
		request.Mode = &mode
	}
	if uid, ok := in.GetUID(); ok {
		request.Uid = &uid
	}
	if gid, ok := in.GetGID(); ok {
		request.Gid = &gid
	}
	if size, ok := in.GetSize(); ok {
		request.Size = &size
	}
	if accessTime, ok := in.GetATime(); ok {
		value := accessTime.UnixNano()
		request.AccessTime = &value
	}
	if modifyTime, ok := in.GetMTime(); ok {
		value := modifyTime.UnixNano()
		request.ModifyTime = &value
	}
	inode, err := n.client.Setattr(ctx, request)
	if err != nil {
		return errnoOf(err)
	}
	fillAttribute(inode, &out.Attr)
	return fs.OK
}

func (n *node) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	entries, err := n.client.Readdir(ctx, n.number())
	if err != nil {
		return nil, errnoOf(err)
	}
	list := make([]fuse.DirEntry, 0, len(entries))
	for _, entry := range entries {
		list = append(list, fuse.DirEntry{Name: entry.Name, Ino: entry.Inode.InodeNumber, Mode: entry.Inode.Mode})
	}
	return fs.NewListDirStream(list), fs.OK
}

func (n *node) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	if flags&(syscall.O_WRONLY|syscall.O_RDWR) == noMatchingFlagBits {
		if n.number() == unregisteredInodeNumber {
			return nil, fuse.FOPEN_DIRECT_IO, fs.OK // reads return EAGAIN until the write is in the index
		}
		return &readSession{client: n.client, inodeNumber: n.number()}, fuse.FOPEN_DIRECT_IO, fs.OK
	}
	if n.number() == unregisteredInodeNumber {
		return nil, noOpenFlags, syscall.ENOTSUP // a file being written cannot be opened and written separately
	}
	inode, err := n.client.Getattr(ctx, n.number())
	if err != nil {
		return nil, noOpenFlags, errnoOf(err)
	}
	if flags&syscall.O_APPEND != noMatchingFlagBits && inode.Size != emptyLength {
		// Appending is a partial update and unsupported. Rejecting at WRITE becomes EIO in the kernel, so reject at open(2)
		return nil, noOpenFlags, syscall.ENOTSUP
	}
	target := &protocol.WriteTarget{ParentInode: inode.ParentInodeNumber, Name: inode.Name, Mode: inode.Mode,
		Uid: inode.UID, Gid: inode.GID}
	session := startWriteSession(n.client, target)
	// The Linux kernel does not put O_TRUNC in OPEN; it sends a size-0 SETATTR later.
	// A non-empty file cannot be written until that arrives (partial updates are unsupported)
	session.truncateRequired = flags&syscall.O_TRUNC == noMatchingFlagBits && inode.Size != emptyLength
	if session.truncateRequired {
		n.addTruncator(ctx, session)
	}
	return session, fuse.FOPEN_DIRECT_IO, fs.OK
}

func (n *node) Read(ctx context.Context, handle fs.FileHandle, destination []byte, offset int64) (fuse.ReadResult, syscall.Errno) {
	if session, ok := handle.(*readSession); ok {
		return session.read(destination, offset)
	}
	if n.number() == unregisteredInodeNumber {
		return nil, syscall.EAGAIN
	}
	data, err := n.client.Read(ctx, n.number(), offset, int64(len(destination)))
	if err != nil {
		return nil, errnoOf(err)
	}
	return fuse.ReadResultData(data), fs.OK
}

func (n *node) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	if err := yaagwadb.ValidateName(name); err != nil {
		return nil, nil, noOpenFlags, errnoOf(err)
	}
	uid, gid := caller(ctx)
	target := &protocol.WriteTarget{ParentInode: n.number(), Name: name,
		Mode: yaagwadb.ModeRegular | mode&^yaagwadb.ModeTypeMask, Uid: uid, Gid: gid}
	session := startWriteSession(n.client, target)
	child := &node{client: n.client, session: session}
	session.fillAttribute(&out.Attr)
	return n.NewInode(ctx, child, fs.StableAttr{Mode: yaagwadb.ModeRegular}), session, fuse.FOPEN_DIRECT_IO, fs.OK
}

func (n *node) Mkdir(ctx context.Context, name string, mode uint32, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	uid, gid := caller(ctx)
	inode, err := n.client.Mkdir(ctx, n.number(), name, mode, uid, gid)
	if err != nil {
		return nil, errnoOf(err)
	}
	fillAttribute(inode, &out.Attr)
	return n.child(ctx, inode), fs.OK
}

func (n *node) Unlink(ctx context.Context, name string) syscall.Errno {
	return errnoOf(n.client.Unlink(ctx, n.number(), name))
}

func (n *node) Rmdir(ctx context.Context, name string) syscall.Errno {
	return errnoOf(n.client.Rmdir(ctx, n.number(), name))
}

func (n *node) Rename(ctx context.Context, name string, newParent fs.InodeEmbedder, newName string, flags uint32) syscall.Errno {
	if flags != noRenameFlags {
		return syscall.ENOTSUP // RENAME_NOREPLACE / RENAME_EXCHANGE are unsupported
	}
	target, ok := newParent.(*node)
	if !ok || target.number() == unregisteredInodeNumber {
		return syscall.EINVAL
	}
	return errnoOf(n.client.Rename(ctx, n.number(), name, target.number(), newName))
}

func (n *node) Link(ctx context.Context, target fs.InodeEmbedder, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	return nil, syscall.EPERM
}

func (n *node) Symlink(ctx context.Context, target, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	return nil, syscall.EPERM
}

func (n *node) Getxattr(ctx context.Context, name string, destination []byte) (uint32, syscall.Errno) {
	if n.number() == unregisteredInodeNumber {
		if n.session == nil {
			return emptyLength, syscall.EAGAIN
		}
		value, exists := n.session.extendedAttribute(name)
		if !exists {
			return emptyLength, noAttribute
		}
		return copyReply(destination, value)
	}
	value, err := n.client.Getxattr(ctx, n.number(), name)
	if err != nil {
		if yaagwadb.Has(err, yaagwadb.NotFound) {
			return emptyLength, noAttribute
		}
		return emptyLength, errnoOf(err)
	}
	return copyReply(destination, value)
}

func copyReply(destination, value []byte) (uint32, syscall.Errno) {
	if len(destination) == emptyLength {
		return uint32(len(value)), fs.OK // size query
	}
	if len(destination) < len(value) {
		return uint32(len(value)), syscall.ERANGE
	}
	return uint32(copy(destination, value)), fs.OK
}

func (n *node) Setxattr(ctx context.Context, name string, data []byte, flags uint32) syscall.Errno {
	if n.number() == unregisteredInodeNumber {
		if n.session == nil {
			return syscall.EAGAIN
		}
		return n.session.setExtendedAttribute(ctx, name, data)
	}
	return errnoOf(n.client.Setxattr(ctx, n.number(), name, data))
}

func (n *node) Listxattr(ctx context.Context, destination []byte) (uint32, syscall.Errno) {
	var names []string
	if n.number() == unregisteredInodeNumber {
		if n.session != nil {
			names = n.session.extendedAttributeNames()
		}
	} else {
		var err error
		if names, err = n.client.Listxattr(ctx, n.number()); err != nil {
			return emptyLength, errnoOf(err)
		}
	}
	var list []byte
	for _, name := range names {
		list = append(append(list, name...), extendedAttributeNameTerminator)
	}
	return copyReply(destination, list)
}

func (n *node) Removexattr(ctx context.Context, name string) syscall.Errno {
	if n.number() == unregisteredInodeNumber {
		if n.session == nil {
			return syscall.EAGAIN
		}
		return n.session.removeExtendedAttribute(name)
	}
	err := n.client.Removexattr(ctx, n.number(), name)
	if yaagwadb.Has(err, yaagwadb.NotFound) {
		return noAttribute
	}
	return errnoOf(err)
}

func (n *node) Statfs(ctx context.Context, out *fuse.StatfsOut) syscall.Errno {
	result, err := n.client.Statfs(ctx)
	if err != nil {
		return errnoOf(err)
	}
	out.Bsize = uint32(result.BlockSize)
	out.Frsize = uint32(result.BlockSize)
	out.Blocks = result.Blocks
	out.Bfree = result.BlocksFree
	out.Bavail = result.BlocksAvailable
	out.Files = result.Files
	out.Ffree = result.FilesFree
	out.NameLen = result.NameLength
	return fs.OK
}

// readSession reads a file opened for reading through one Read stream from the daemon.
// The kernel sends the reads of a sequential read(2) one after another, each up to max_read,
// so following them on one stream avoids starting a Read per request and lets the stream send ahead.
// A read at another offset than where the stream is ends it and starts a new one there.
type readSession struct {
	client      *yaagwadb.Client
	inodeNumber uint64

	mutex    sync.Mutex
	reader   *yaagwadb.FileDataReader // nil until the first read, and after an error
	position int64                    // offset the next byte of reader is at
}

var (
	_ fs.FileReleaser = (*readSession)(nil)
)

func (r *readSession) read(destination []byte, offset int64) (fuse.ReadResult, syscall.Errno) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.reader == nil || offset != r.position {
		r.closeReader()
		// The stream outlives the request, so it does not use the request's context
		reader, err := r.client.OpenReader(context.Background(), r.inodeNumber, offset)
		if err != nil {
			return nil, errnoOf(err)
		}
		r.reader, r.position = reader, offset
	}
	length, err := io.ReadFull(r.reader, destination)
	r.position += int64(length)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		r.closeReader()
		return nil, errnoOf(err)
	}
	return fuse.ReadResultData(destination[:length]), fs.OK
}

func (r *readSession) Release(ctx context.Context) syscall.Errno {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.closeReader()
	return fs.OK
}

// closeReader ends the stream, if any. Called with r.mutex held.
func (r *readSession) closeReader() {
	if r.reader != nil {
		r.reader.Close()
		r.reader = nil
	}
}

// writeSession streams the write from create (or open with O_TRUNC) to release
// to the daemon as a single gRPC Write stream. Writes must be in offset order.
// A file being written is not yet in the index and has no inode number, so attribute and xattr changes are accepted and held here,
// then sent to the daemon once the Write response provides the inode.
type writeSession struct {
	mutex    sync.Mutex
	client   *yaagwadb.Client
	pipe     *io.PipeWriter
	written  int64
	target   *protocol.WriteTarget
	started  time.Time
	finished chan struct{}
	result   error
	response *protocol.WriteResponse // read only after finished is closed
	closed   bool
	onClose  func() // called once when the write is closed, if set

	inodeNumber        uint64 // set once the Write is in the index; changes are held until then
	truncateRequired   bool   // a non-empty file was opened without O_TRUNC, so waiting for a size-0 SETATTR
	mode               uint32 // mode through modifyTime are attributes with accepted changes applied
	uid, gid           uint32
	accessTime         time.Time
	modifyTime         time.Time
	pendingAttributes  *protocol.SetattrRequest // attribute changes not yet sent to the daemon
	extendedAttributes map[string][]byte        // xattrs set on this file
	pendingNames       []string                 // names of xattrs not yet sent to the daemon
}

var (
	_ fs.FileWriter    = (*writeSession)(nil)
	_ fs.FileFlusher   = (*writeSession)(nil)
	_ fs.FileReleaser  = (*writeSession)(nil)
	_ fs.FileSetattrer = (*writeSession)(nil)
)

func startWriteSession(client *yaagwadb.Client, target *protocol.WriteTarget) *writeSession {
	reader, writer := io.Pipe()
	started := time.Now()
	session := &writeSession{client: client, pipe: writer, target: target, started: started, finished: make(chan struct{}),
		mode: target.Mode, uid: target.Uid, gid: target.Gid, accessTime: started, modifyTime: started,
		pendingAttributes: &protocol.SetattrRequest{}, extendedAttributes: map[string][]byte{}}
	go func() {
		response, err := client.Write(context.Background(), target, reader)
		reader.CloseWithError(err) // if the daemon responds early, fail subsequent writes
		session.response, session.result = response, err
		close(session.finished)
	}()
	return session
}

func (w *writeSession) fillAttribute(out *fuse.Attr) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	out.Size = uint64(w.written)
	out.Mode = w.mode
	out.Nlink = regularFileLinkCount
	out.Owner = fuse.Owner{Uid: w.uid, Gid: w.gid}
	out.Atime, out.Mtime, out.Ctime = uint64(w.accessTime.Unix()), uint64(w.modifyTime.Unix()), uint64(w.started.Unix())
}

func (w *writeSession) Write(ctx context.Context, data []byte, offset int64) (uint32, syscall.Errno) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.closed {
		return emptyLength, syscall.EBADF
	}
	if offset != w.written || w.truncateRequired {
		return emptyLength, syscall.ENOTSUP // seek writes and partial updates are unsupported
	}
	n, err := w.pipe.Write(data)
	w.written += int64(n)
	if err != nil {
		return uint32(n), errnoOf(err)
	}
	return uint32(n), fs.OK
}

func (w *writeSession) Flush(ctx context.Context) syscall.Errno {
	w.mutex.Lock()
	if !w.closed && w.written == emptyLength {
		// Shell redirection and the like close a duplicated file descriptor, so FLUSH arrives before any write.
		// If nothing has been written yet, keep waiting instead of closing (an empty file is closed on RELEASE)
		w.mutex.Unlock()
		return fs.OK
	}
	return w.finish(ctx)
}

func (w *writeSession) Release(ctx context.Context) syscall.Errno {
	w.mutex.Lock()
	return w.finish(ctx)
}

func (w *writeSession) finish(ctx context.Context) syscall.Errno {
	// Called with w.mutex held; released here
	if w.truncateRequired {
		// Nothing was written or truncated. Cancel the write to keep the original file
		closing := !w.closed
		if closing {
			w.closed = true
			w.pipe.CloseWithError(errors.New(WriteCancelled))
		}
		w.mutex.Unlock()
		w.afterClose(closing)
		<-w.finished
		return fs.OK
	}
	closing := !w.closed
	if closing {
		w.closed = true
		w.pipe.Close()
	}
	w.mutex.Unlock()
	w.afterClose(closing)
	<-w.finished
	if w.result != nil {
		return errnoOf(w.result)
	}
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.response.Attributes != nil {
		w.inodeNumber = w.response.Attributes.Inode
	}
	return w.sendPending(ctx)
}

// afterClose calls onClose if this call closed the write. Called without w.mutex held.
func (w *writeSession) afterClose(closing bool) {
	if closing && w.onClose != nil {
		w.onClose()
	}
}

func (w *writeSession) Setattr(ctx context.Context, in *fuse.SetAttrIn, out *fuse.AttrOut) syscall.Errno {
	errno := w.setAttribute(ctx, in)
	w.fillAttribute(&out.Attr)
	return errno
}

func (w *writeSession) setAttribute(ctx context.Context, in *fuse.SetAttrIn) syscall.Errno {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if size, ok := in.GetSize(); ok {
		if size != uint64(w.written) {
			return syscall.ENOTSUP // truncating or extending during a write is unsupported
		}
		w.truncateRequired = false
	}
	if mode, ok := in.GetMode(); ok {
		w.mode = yaagwadb.ModeRegular | mode&^yaagwadb.ModeTypeMask
		w.pendingAttributes.Mode = &mode
	}
	if uid, ok := in.GetUID(); ok {
		w.uid = uid
		w.pendingAttributes.Uid = &uid
	}
	if gid, ok := in.GetGID(); ok {
		w.gid = gid
		w.pendingAttributes.Gid = &gid
	}
	if accessTime, ok := in.GetATime(); ok {
		w.accessTime = accessTime
		value := accessTime.UnixNano()
		w.pendingAttributes.AccessTime = &value
	}
	if modifyTime, ok := in.GetMTime(); ok {
		w.modifyTime = modifyTime
		value := modifyTime.UnixNano()
		w.pendingAttributes.ModifyTime = &value
	}
	return w.sendPending(ctx)
}

func (w *writeSession) registeredInodeNumber() uint64 {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.inodeNumber
}

func (w *writeSession) extendedAttribute(name string) ([]byte, bool) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	value, exists := w.extendedAttributes[name]
	return value, exists
}

func (w *writeSession) extendedAttributeNames() []string {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	names := make([]string, 0, len(w.extendedAttributes))
	for name := range w.extendedAttributes {
		names = append(names, name)
	}
	return names
}

func (w *writeSession) setExtendedAttribute(ctx context.Context, name string, data []byte) syscall.Errno {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if _, pending := w.pendingIndex(name); !pending {
		w.pendingNames = append(w.pendingNames, name)
	}
	w.extendedAttributes[name] = append([]byte(nil), data...) // data is a FUSE buffer reused after the call
	return w.sendPending(ctx)
}

func (w *writeSession) removeExtendedAttribute(name string) syscall.Errno {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if _, exists := w.extendedAttributes[name]; !exists {
		return noAttribute
	}
	index, pending := w.pendingIndex(name)
	if !pending {
		return syscall.ENOTSUP // removal after sending to the daemon is unsupported
	}
	w.pendingNames = append(w.pendingNames[:index], w.pendingNames[index+1:]...)
	delete(w.extendedAttributes, name)
	return fs.OK
}

func (w *writeSession) pendingIndex(name string) (int, bool) {
	for index, pendingName := range w.pendingNames {
		if pendingName == name {
			return index, true
		}
	}
	return len(w.pendingNames), false
}

func (w *writeSession) sendPending(ctx context.Context) syscall.Errno {
	// Called with w.mutex held. Keep changes held until the inode is known
	if w.inodeNumber == unregisteredInodeNumber {
		return fs.OK
	}
	if attributes := w.pendingAttributes; attributes.Mode != nil || attributes.Uid != nil || attributes.Gid != nil ||
		attributes.AccessTime != nil || attributes.ModifyTime != nil {
		attributes.Inode = w.inodeNumber
		w.pendingAttributes = &protocol.SetattrRequest{}
		if _, err := w.client.Setattr(ctx, attributes); err != nil {
			return errnoOf(err)
		}
	}
	for len(w.pendingNames) > 0 {
		name := w.pendingNames[0]
		w.pendingNames = w.pendingNames[1:]
		if err := w.client.Setxattr(ctx, w.inodeNumber, name, w.extendedAttributes[name]); err != nil {
			return errnoOf(err)
		}
	}
	return fs.OK
}
