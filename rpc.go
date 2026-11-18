package yaagwadb

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"os"

	"github.com/pkg/errors"

	"yaagwadb/protocol"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	errorDomain       = "yaagwadb"
	streamPieceLength = 1 << 20 // size of one message when streaming file data bytes
	// defaultTaskListLimit is the per-queue count for ListTasks when no limit is given.
	defaultTaskListLimit = 50
	// taskListLimitUnspecified means no limit was given to ListTasks.
	taskListLimitUnspecified = 0
)

// rpcWithoutErrorMessage means the error contains no error message constant (ErrorInfo.reason is empty).
const rpcWithoutErrorMessage = ""

const (
	grpcSocketNetwork      = "unix"
	grpcSocketTargetScheme = "unix:"
	grpcSocketPermission   = 0o600
)

const (
	regularFileLinkCount  = 1
	directoryLinkCount    = 2
	attributeDeviceNumber = 0
	statBlockUnit         = 512
	// statBlockRoundUp is added to Size to round it up to a multiple of statBlockUnit.
	statBlockRoundUp = statBlockUnit - 1
)

const (
	supportedTruncateSize = 0
	readUntilEndSize      = 0
)

const (
	rpcStatusMessageFormat    = "%w: %s"
	rpcTruncateFormat         = "%w: truncate to %d bytes"
	rpcReadRangeFormat        = "%w: offset %d size %d"
	rpcWriteTargetTwiceFormat = "%w: write target sent twice"
	rpcWriteTargetFirstFormat = "%w: first write message must be the target"
	rpcWriteEndedEarlyFormat  = "%w: the daemon ended the write before the content ended"
)

const (
	// rpcInternalErrorMessage is sent to the client in place of the text of an error without a constant,
	// because that text can name paths in the data directory. The text itself is in the daemon log
	rpcInternalErrorMessage = "internal error (details in the daemon log)"
	rpcLogInternalError     = "internal error"
	rpcLogServeFailed       = "grpc serve"
	rpcLogKeyError          = "error"
	rpcLogKeyStack          = "stack"
)

// statusCodes maps error message constants to gRPC codes.
var statusCodes = map[string]codes.Code{
	NotFound:        codes.NotFound,
	Corrupted:       codes.DataLoss,
	Exists:          codes.AlreadyExists,
	NotDirectory:    codes.FailedPrecondition,
	IsDirectory:     codes.FailedPrecondition,
	NotEmpty:        codes.FailedPrecondition,
	NotSupported:    codes.Unimplemented,
	InvalidArgument: codes.InvalidArgument,
	NoSpace:         codes.ResourceExhausted,
	Closed:          codes.Unavailable,
	Locked:          codes.FailedPrecondition,
}

func toStatus(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok && messageOf(err) == rpcWithoutErrorMessage {
		return err
	}
	if cancelled(err) {
		return status.FromContextError(err).Err()
	}
	message := messageOf(err)
	code, ok := statusCodes[message]
	if !ok {
		code = codes.Internal
	}
	text := err.Error()
	if message == rpcWithoutErrorMessage {
		text = rpcInternalErrorMessage
	}
	result := status.New(code, text)
	if message != rpcWithoutErrorMessage {
		if detailed, detailError := result.WithDetails(&errdetails.ErrorInfo{Reason: message, Domain: errorDomain}); detailError == nil {
			result = detailed
		}
	}
	return result.Err()
}

func fromStatus(err error) error {
	if err == nil {
		return nil
	}
	result, ok := status.FromError(err)
	if !ok {
		return err
	}
	for _, detail := range result.Details() {
		if information, ok := detail.(*errdetails.ErrorInfo); ok && information.Domain == errorDomain && information.Reason != rpcWithoutErrorMessage {
			return fmt.Errorf(rpcStatusMessageFormat, errors.New(information.Reason), result.Message())
		}
	}
	return err
}

type server struct {
	protocol.UnimplementedYaagwaDBServer
	daemon *Daemon
}

func (s *server) toStatus(err error) error {
	// A call the client ended is not a daemon failure, so it is not logged
	if err != nil && messageOf(err) == rpcWithoutErrorMessage && !cancelled(err) {
		s.daemon.logger.Error(rpcLogInternalError, rpcLogKeyError, err, rpcLogKeyStack, StackOf(err))
	}
	return toStatus(err)
}

// cancelled reports whether err comes from the client ending the call (cancellation or its deadline).
func cancelled(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	code := status.Code(err)
	return code == codes.Canceled || code == codes.DeadlineExceeded
}

func ListenGRPC(daemon *Daemon, socket string) (*grpc.Server, error) {
	if err := os.Remove(socket); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	listener, err := net.Listen(grpcSocketNetwork, socket)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(socket, grpcSocketPermission); err != nil {
		listener.Close()
		return nil, err
	}
	grpcServer := NewGRPCServer(daemon)
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			daemon.logger.Error(rpcLogServeFailed, rpcLogKeyError, err)
		}
	}()
	return grpcServer, nil
}

func NewGRPCServer(daemon *Daemon) *grpc.Server {
	grpcServer := grpc.NewServer()
	protocol.RegisterYaagwaDBServer(grpcServer, &server{daemon: daemon})
	return grpcServer
}

func toAttributes(inode *Inode) *protocol.Attributes {
	nlink := uint32(regularFileLinkCount)
	if inode.IsDirectory() {
		nlink = directoryLinkCount
	}
	return &protocol.Attributes{
		Inode: inode.InodeNumber, ParentInode: inode.ParentInodeNumber, Name: inode.Name,
		Mode: inode.Mode, Uid: inode.UID, Gid: inode.GID, Size: inode.Size,
		AccessTime: inode.AccessTime, ModifyTime: inode.ModifyTime, ChangeTime: inode.ChangeTime,
		Nlink: nlink, Rdev: attributeDeviceNumber, Blocks: (inode.Size + statBlockRoundUp) / statBlockUnit, BlockSize: BlockSize,
	}
}

func fromAttributes(attributes *protocol.Attributes) *Inode {
	return &Inode{
		InodeNumber: attributes.Inode, ParentInodeNumber: attributes.ParentInode, Name: attributes.Name,
		Mode: attributes.Mode, UID: attributes.Uid, GID: attributes.Gid, Size: attributes.Size,
		AccessTime: attributes.AccessTime, ModifyTime: attributes.ModifyTime, ChangeTime: attributes.ChangeTime,
	}
}

func (s *server) Lookup(_ context.Context, request *protocol.LookupRequest) (*protocol.Attributes, error) {
	inode, err := s.daemon.Lookup(request.ParentInode, request.Name)
	if err != nil {
		return nil, s.toStatus(err)
	}
	return toAttributes(inode), nil
}

func (s *server) Getattr(_ context.Context, request *protocol.GetattrRequest) (*protocol.Attributes, error) {
	inode, err := s.daemon.GetAttribute(request.Inode)
	if err != nil {
		return nil, s.toStatus(err)
	}
	return toAttributes(inode), nil
}

func (s *server) Setattr(_ context.Context, request *protocol.SetattrRequest) (*protocol.Attributes, error) {
	if request.Size != nil {
		if *request.Size != supportedTruncateSize {
			return nil, s.toStatus(fmt.Errorf(rpcTruncateFormat, errors.New(NotSupported), *request.Size))
		}
		if err := s.daemon.Truncate(request.Inode); err != nil {
			return nil, s.toStatus(err)
		}
	}
	var mask AttributeMask
	var attribute Attribute
	if request.Mode != nil {
		mask |= AttributeMode
		attribute.Mode = *request.Mode
	}
	if request.Uid != nil {
		mask |= AttributeUID
		attribute.UID = *request.Uid
	}
	if request.Gid != nil {
		mask |= AttributeGID
		attribute.GID = *request.Gid
	}
	if request.AccessTime != nil {
		mask |= AttributeAccessTime
		attribute.AccessTime = *request.AccessTime
	}
	if request.ModifyTime != nil {
		mask |= AttributeModifyTime
		attribute.ModifyTime = *request.ModifyTime
	}
	inode, err := s.daemon.Setattr(request.Inode, mask, attribute)
	if err != nil {
		return nil, s.toStatus(err)
	}
	return toAttributes(inode), nil
}

func (s *server) Readdir(request *protocol.ReaddirRequest, stream grpc.ServerStreamingServer[protocol.DirectoryEntry]) error {
	entries, err := s.daemon.ListDirectory(request.Inode)
	if err != nil {
		return s.toStatus(err)
	}
	for _, entry := range entries {
		if err := stream.Send(&protocol.DirectoryEntry{Name: entry.Name, Attributes: toAttributes(&entry.Inode)}); err != nil {
			return err
		}
	}
	return nil
}

func (s *server) Read(request *protocol.ReadRequest, stream grpc.ServerStreamingServer[protocol.ReadResponse]) error {
	if request.Offset < 0 || request.Size < 0 {
		return s.toStatus(fmt.Errorf(rpcReadRangeFormat, errors.New(InvalidArgument), request.Offset, request.Size))
	}
	file, err := s.daemon.OpenFileData(request.Inode)
	if err != nil {
		return s.toStatus(err)
	}
	defer file.Close()
	var reader io.Reader = io.NewSectionReader(file, request.Offset, math.MaxInt64-request.Offset)
	if request.Size > readUntilEndSize {
		reader = io.LimitReader(reader, request.Size)
	}
	// The buffer is allocated on every call, so a small read gets a small buffer instead of a whole piece.
	// A read to the end is sized by what is left of the file
	remaining := request.Size
	if remaining == readUntilEndSize {
		information, err := file.Stat()
		if err != nil {
			return s.toStatus(err)
		}
		remaining = information.Size() - request.Offset
	}
	if remaining <= readUntilEndSize {
		return nil // nothing to send at or past the end
	}
	buffer := make([]byte, min(remaining, streamPieceLength))
	for {
		length, err := io.ReadFull(reader, buffer)
		if length > 0 {
			if sendError := stream.Send(&protocol.ReadResponse{Data: buffer[:length]}); sendError != nil {
				return sendError
			}
		}
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil
		}
		if err != nil {
			return s.toStatus(err)
		}
	}
}

// writeStreamReader reads the second and later messages of the Write client stream as an io.Reader.
type writeStreamReader struct {
	stream  grpc.ClientStreamingServer[protocol.WriteRequest, protocol.WriteResponse]
	current []byte
}

func (r *writeStreamReader) Read(p []byte) (int, error) {
	for len(r.current) == 0 {
		message, err := r.stream.Recv()
		if err != nil {
			return 0, err // normal end is io.EOF
		}
		data, ok := message.Content.(*protocol.WriteRequest_Data)
		if !ok {
			return 0, fmt.Errorf(rpcWriteTargetTwiceFormat, errors.New(InvalidArgument))
		}
		r.current = data.Data
	}
	length := copy(p, r.current)
	r.current = r.current[length:]
	return length, nil
}

func (s *server) Write(stream grpc.ClientStreamingServer[protocol.WriteRequest, protocol.WriteResponse]) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	target := first.GetTarget()
	if target == nil {
		return s.toStatus(fmt.Errorf(rpcWriteTargetFirstFormat, errors.New(InvalidArgument)))
	}
	inode, err := s.daemon.Write(target.ParentInode, target.Name, target.Mode, target.Uid, target.Gid,
		&writeStreamReader{stream: stream})
	if err != nil {
		return s.toStatus(err)
	}
	return stream.SendAndClose(&protocol.WriteResponse{FileDataId: inode.FileDataID, Size: inode.Size, Attributes: toAttributes(inode)})
}

func (s *server) Unlink(_ context.Context, request *protocol.UnlinkRequest) (*protocol.Empty, error) {
	return &protocol.Empty{}, s.toStatus(s.daemon.Unlink(request.ParentInode, request.Name))
}

func (s *server) Mkdir(_ context.Context, request *protocol.MkdirRequest) (*protocol.Attributes, error) {
	inode, err := s.daemon.Mkdir(request.ParentInode, request.Name, request.Mode, request.Uid, request.Gid)
	if err != nil {
		return nil, s.toStatus(err)
	}
	return toAttributes(inode), nil
}

func (s *server) Rmdir(_ context.Context, request *protocol.RmdirRequest) (*protocol.Empty, error) {
	return &protocol.Empty{}, s.toStatus(s.daemon.Rmdir(request.ParentInode, request.Name))
}

func (s *server) Rename(_ context.Context, request *protocol.RenameRequest) (*protocol.Empty, error) {
	err := s.daemon.Rename(request.OldParentInode, request.OldName, request.NewParentInode, request.NewName)
	return &protocol.Empty{}, s.toStatus(err)
}

func (s *server) Getxattr(_ context.Context, request *protocol.GetxattrRequest) (*protocol.GetxattrResponse, error) {
	value, err := s.daemon.GetExtendedAttribute(request.Inode, request.Name)
	if err != nil {
		return nil, s.toStatus(err)
	}
	return &protocol.GetxattrResponse{Value: value}, nil
}

func (s *server) Setxattr(_ context.Context, request *protocol.SetxattrRequest) (*protocol.Empty, error) {
	return &protocol.Empty{}, s.toStatus(s.daemon.SetExtendedAttribute(request.Inode, request.Name, request.Value))
}

func (s *server) Listxattr(_ context.Context, request *protocol.ListxattrRequest) (*protocol.ListxattrResponse, error) {
	names, err := s.daemon.ListExtendedAttributes(request.Inode)
	if err != nil {
		return nil, s.toStatus(err)
	}
	return &protocol.ListxattrResponse{Names: names}, nil
}

func (s *server) Removexattr(_ context.Context, request *protocol.RemovexattrRequest) (*protocol.Empty, error) {
	return &protocol.Empty{}, s.toStatus(s.daemon.RemoveExtendedAttribute(request.Inode, request.Name))
}

func (s *server) Statfs(context.Context, *protocol.Empty) (*protocol.StatfsResponse, error) {
	result, err := s.daemon.Statfs()
	if err != nil {
		return nil, s.toStatus(err)
	}
	return &protocol.StatfsResponse{
		BlockSize: result.BlockSize, Blocks: result.Blocks, BlocksFree: result.BlocksFree,
		BlocksAvailable: result.BlocksAvailable, Files: result.Files, FilesFree: result.FilesFree,
		NameLength: maximumNameLength,
	}, nil
}

func (s *server) Query(ctx context.Context, request *protocol.QueryRequest) (*protocol.QueryResponse, error) {
	parameters := make([]QueryValue, 0, len(request.Parameters))
	for _, parameter := range request.Parameters {
		parameters = append(parameters, fromProtocolValue(parameter))
	}
	named := make(map[string]QueryValue, len(request.NamedParameters))
	for _, parameter := range request.NamedParameters {
		named[parameter.Name] = fromProtocolValue(parameter.Value)
	}
	result, err := s.daemon.Query(ctx, request.Sql, parameters, named)
	if err != nil {
		return nil, s.toStatus(err)
	}
	return toProtocolResult(result), nil
}

func fromProtocolValue(value *protocol.QueryValue) QueryValue {
	switch typed := value.GetValue().(type) {
	case *protocol.QueryValue_Integer:
		return QueryValue{Integer: &typed.Integer}
	case *protocol.QueryValue_Real:
		return QueryValue{Real: &typed.Real}
	case *protocol.QueryValue_Text:
		return QueryValue{Text: &typed.Text}
	}
	return QueryValue{}
}

func toProtocolValue(value QueryValue) *protocol.QueryValue {
	switch {
	case value.Integer != nil:
		return &protocol.QueryValue{Value: &protocol.QueryValue_Integer{Integer: *value.Integer}}
	case value.Real != nil:
		return &protocol.QueryValue{Value: &protocol.QueryValue_Real{Real: *value.Real}}
	case value.Text != nil:
		return &protocol.QueryValue{Value: &protocol.QueryValue_Text{Text: *value.Text}}
	}
	return &protocol.QueryValue{}
}

func toProtocolResult(result *QueryResult) *protocol.QueryResponse {
	response := &protocol.QueryResponse{Columns: result.Columns}
	for _, row := range result.Rows {
		values := make([]*protocol.QueryValue, 0, len(row))
		for _, value := range row {
			values = append(values, toProtocolValue(value))
		}
		response.Rows = append(response.Rows, &protocol.QueryRow{Values: values})
	}
	return response
}

func fromProtocolResult(response *protocol.QueryResponse) *QueryResult {
	result := &QueryResult{Columns: response.Columns}
	for _, row := range response.Rows {
		values := make([]QueryValue, 0, len(row.Values))
		for _, value := range row.Values {
			values = append(values, fromProtocolValue(value))
		}
		result.Rows = append(result.Rows, values)
	}
	return result
}

func (s *server) Status(context.Context, *protocol.Empty) (*protocol.StatusResponse, error) {
	result, err := s.daemon.Status()
	if err != nil {
		return nil, s.toStatus(err)
	}
	response := &protocol.StatusResponse{FileData: result.FileData, LogicallyDeleted: result.LogicallyDeleted}
	for _, queue := range result.Queues {
		response.Queues = append(response.Queues, &protocol.QueueStatus{
			Queue: queue.Queue, Pending: queue.Pending, Running: queue.Running, Failed: queue.Failed,
		})
	}
	return response, nil
}

func (s *server) ListTasks(_ context.Context, request *protocol.ListTasksRequest) (*protocol.ListTasksResponse, error) {
	limit := int(request.Limit)
	if limit == taskListLimitUnspecified {
		limit = defaultTaskListLimit
	}
	tasks, err := s.daemon.ListTasks(request.Queue, request.ActiveOnly, limit)
	if err != nil {
		return nil, s.toStatus(err)
	}
	response := &protocol.ListTasksResponse{}
	for _, task := range tasks {
		response.Tasks = append(response.Tasks, &protocol.Task{
			Queue: task.Queue, Id: task.ID, Kind: TaskKindName(task.Queue, task.Kind), State: task.State.String(),
			Attempts: task.Attempts, NextRunAt: task.NextRunAt, LastError: task.LastError,
			CreatedAt: task.CreatedAt, FinishedAt: task.FinishedAt,
		})
	}
	return response, nil
}

// Client is the daemon's gRPC client. Errors are converted back so Has can match their constants.
type Client struct {
	connection *grpc.ClientConn
	service    protocol.YaagwaDBClient
}

func Dial(socket string) (*Client, error) {
	connection, err := grpc.NewClient(grpcSocketTargetScheme+socket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{connection: connection, service: protocol.NewYaagwaDBClient(connection)}, nil
}

func (c *Client) Close() error { return c.connection.Close() }

func (c *Client) Service() protocol.YaagwaDBClient { return c.service }

func (c *Client) Lookup(ctx context.Context, parent uint64, name string) (*Inode, error) {
	attributes, err := c.service.Lookup(ctx, &protocol.LookupRequest{ParentInode: parent, Name: name})
	if err != nil {
		return nil, fromStatus(err)
	}
	return fromAttributes(attributes), nil
}

func (c *Client) Getattr(ctx context.Context, inodeNumber uint64) (*Inode, error) {
	attributes, err := c.service.Getattr(ctx, &protocol.GetattrRequest{Inode: inodeNumber})
	if err != nil {
		return nil, fromStatus(err)
	}
	return fromAttributes(attributes), nil
}

func (c *Client) Setattr(ctx context.Context, request *protocol.SetattrRequest) (*Inode, error) {
	attributes, err := c.service.Setattr(ctx, request)
	if err != nil {
		return nil, fromStatus(err)
	}
	return fromAttributes(attributes), nil
}

func (c *Client) Readdir(ctx context.Context, inodeNumber uint64) ([]DirectoryEntry, error) {
	stream, err := c.service.Readdir(ctx, &protocol.ReaddirRequest{Inode: inodeNumber})
	if err != nil {
		return nil, fromStatus(err)
	}
	var entries []DirectoryEntry
	for {
		entry, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return entries, nil
		}
		if err != nil {
			return nil, fromStatus(err)
		}
		entries = append(entries, DirectoryEntry{Name: entry.Name, Inode: *fromAttributes(entry.Attributes)})
	}
}

func (c *Client) Read(ctx context.Context, inodeNumber uint64, offset, size int64) ([]byte, error) {
	stream, err := c.service.Read(ctx, &protocol.ReadRequest{Inode: inodeNumber, Offset: offset, Size: size})
	if err != nil {
		return nil, fromStatus(err)
	}
	var data []byte
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return data, nil
		}
		if err != nil {
			return nil, fromStatus(err)
		}
		data = append(data, response.Data...)
	}
}

// FileDataReader reads the content of a file from an offset to the end through one Read stream.
// Close ends the stream; a FileDataReader that is not read to the end must be closed.
type FileDataReader struct {
	stream  grpc.ServerStreamingClient[protocol.ReadResponse]
	cancel  context.CancelFunc
	current []byte
}

// OpenReader starts a Read stream of the content from offset to the end.
// An error from the daemon, such as NotFound, is returned by the first Read.
func (c *Client) OpenReader(ctx context.Context, inodeNumber uint64, offset int64) (*FileDataReader, error) {
	ctx, cancel := context.WithCancel(ctx)
	stream, err := c.service.Read(ctx, &protocol.ReadRequest{Inode: inodeNumber, Offset: offset, Size: readUntilEndSize})
	if err != nil {
		cancel()
		return nil, fromStatus(err)
	}
	return &FileDataReader{stream: stream, cancel: cancel}, nil
}

func (r *FileDataReader) Read(p []byte) (int, error) {
	for len(r.current) == 0 {
		response, err := r.stream.Recv()
		if errors.Is(err, io.EOF) {
			return 0, io.EOF
		}
		if err != nil {
			return 0, fromStatus(err)
		}
		r.current = response.Data
	}
	length := copy(p, r.current)
	r.current = r.current[length:]
	return length, nil
}

func (r *FileDataReader) Close() error {
	r.cancel()
	return nil
}

// Write sends target and then content as one Write stream. The daemon answers only when the stream ends,
// or earlier when it gives up on the write; the answer is awaited alongside the content,
// so a failure is returned as soon as the daemon reports it, without waiting for the next piece of the content.
// content is read in its own goroutine, which ends with the next Read after Write returns.
func (c *Client) Write(ctx context.Context, target *protocol.WriteTarget, content io.Reader) (*protocol.WriteResponse, error) {
	// Cancelling on return ends the stream if Write returns before the daemon answered
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := c.service.Write(ctx)
	if err != nil {
		return nil, fromStatus(err)
	}
	response := &protocol.WriteResponse{}
	ended := make(chan error, 1)
	go func() {
		ended <- stream.RecvMsg(response)
	}()
	if err := stream.Send(&protocol.WriteRequest{Content: &protocol.WriteRequest_Target{Target: target}}); err != nil {
		return nil, endedWith(ended, err)
	}
	pieces := make(chan bodyPiece)
	stop := make(chan struct{})
	defer close(stop)
	go readPieces(content, pieces, stop)
	for {
		select {
		case err := <-ended:
			// The daemon answered before the content ended: it gave up on the write
			if err == nil {
				err = fmt.Errorf(rpcWriteEndedEarlyFormat, errors.New(InvalidArgument))
			}
			return nil, fromStatus(err)
		case piece := <-pieces:
			if len(piece.data) > 0 {
				if err := stream.Send(&protocol.WriteRequest{Content: &protocol.WriteRequest_Data{Data: piece.data}}); err != nil {
					return nil, endedWith(ended, err)
				}
			}
			if errors.Is(piece.err, io.EOF) {
				if err := stream.CloseSend(); err != nil {
					return nil, fromStatus(err)
				}
				if err := <-ended; err != nil {
					return nil, fromStatus(err)
				}
				return response, nil
			}
			if piece.err != nil {
				return nil, piece.err
			}
		}
	}
}

// bodyPiece is one Read of the content: a copy of the bytes read and the error Read returned.
type bodyPiece struct {
	data []byte
	err  error
}

// readPieces reads content into pieces until Read returns an error or stop is closed.
func readPieces(content io.Reader, pieces chan<- bodyPiece, stop <-chan struct{}) {
	buffer := make([]byte, streamPieceLength)
	for {
		length, err := content.Read(buffer)
		piece := bodyPiece{data: append([]byte(nil), buffer[:length]...), err: err}
		select {
		case pieces <- piece:
		case <-stop:
			return
		}
		if err != nil {
			return
		}
	}
}

// endedWith returns the daemon's answer when a Send failed because the daemon had already ended the stream.
func endedWith(ended <-chan error, err error) error {
	if errors.Is(err, io.EOF) {
		err = <-ended
	}
	return fromStatus(err)
}

func (c *Client) Unlink(ctx context.Context, parent uint64, name string) error {
	_, err := c.service.Unlink(ctx, &protocol.UnlinkRequest{ParentInode: parent, Name: name})
	return fromStatus(err)
}

func (c *Client) Mkdir(ctx context.Context, parent uint64, name string, mode, uid, gid uint32) (*Inode, error) {
	attributes, err := c.service.Mkdir(ctx, &protocol.MkdirRequest{ParentInode: parent, Name: name, Mode: mode, Uid: uid, Gid: gid})
	if err != nil {
		return nil, fromStatus(err)
	}
	return fromAttributes(attributes), nil
}

func (c *Client) Rmdir(ctx context.Context, parent uint64, name string) error {
	_, err := c.service.Rmdir(ctx, &protocol.RmdirRequest{ParentInode: parent, Name: name})
	return fromStatus(err)
}

func (c *Client) Rename(ctx context.Context, oldParent uint64, oldName string, newParent uint64, newName string) error {
	_, err := c.service.Rename(ctx, &protocol.RenameRequest{OldParentInode: oldParent, OldName: oldName,
		NewParentInode: newParent, NewName: newName})
	return fromStatus(err)
}

func (c *Client) Getxattr(ctx context.Context, inodeNumber uint64, name string) ([]byte, error) {
	response, err := c.service.Getxattr(ctx, &protocol.GetxattrRequest{Inode: inodeNumber, Name: name})
	if err != nil {
		return nil, fromStatus(err)
	}
	return response.Value, nil
}

func (c *Client) Setxattr(ctx context.Context, inodeNumber uint64, name string, value []byte) error {
	_, err := c.service.Setxattr(ctx, &protocol.SetxattrRequest{Inode: inodeNumber, Name: name, Value: value})
	return fromStatus(err)
}

func (c *Client) Listxattr(ctx context.Context, inodeNumber uint64) ([]string, error) {
	response, err := c.service.Listxattr(ctx, &protocol.ListxattrRequest{Inode: inodeNumber})
	if err != nil {
		return nil, fromStatus(err)
	}
	return response.Names, nil
}

func (c *Client) Removexattr(ctx context.Context, inodeNumber uint64, name string) error {
	_, err := c.service.Removexattr(ctx, &protocol.RemovexattrRequest{Inode: inodeNumber, Name: name})
	return fromStatus(err)
}

func (c *Client) Statfs(ctx context.Context) (*protocol.StatfsResponse, error) {
	response, err := c.service.Statfs(ctx, &protocol.Empty{})
	return response, fromStatus(err)
}

func (c *Client) Status(ctx context.Context) (*protocol.StatusResponse, error) {
	response, err := c.service.Status(ctx, &protocol.Empty{})
	return response, fromStatus(err)
}

func (c *Client) ListTasks(ctx context.Context, request *protocol.ListTasksRequest) (*protocol.ListTasksResponse, error) {
	response, err := c.service.ListTasks(ctx, request)
	return response, fromStatus(err)
}

func (c *Client) Query(ctx context.Context, statement string, parameters []QueryValue, named map[string]QueryValue) (*QueryResult, error) {
	request := &protocol.QueryRequest{Sql: statement}
	for _, parameter := range parameters {
		request.Parameters = append(request.Parameters, toProtocolValue(parameter))
	}
	for name, parameter := range named {
		request.NamedParameters = append(request.NamedParameters, &protocol.NamedQueryValue{Name: name, Value: toProtocolValue(parameter)})
	}
	response, err := c.service.Query(ctx, request)
	if err != nil {
		return nil, fromStatus(err)
	}
	return fromProtocolResult(response), nil
}
