package yaagwadb

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/errors"
)

const (
	schemaVersion   = 3
	RootInodeNumber = 1

	ModeTypeMask  uint32 = 0o170000
	ModeDirectory uint32 = 0o040000
	ModeRegular   uint32 = 0o100000

	BlockSize = 4096

	maximumNameLength                   = 255
	MaximumExtendedAttributeValueLength = 64 << 10
)

const (
	metaKeySchemaVersion = "schema_version"
	metaKeyLastWriteAt   = "last_write_at" // time the index was last modified
)

const (
	indexErrorFormatSchemaVersion                  = "%w: schema version %d"
	indexErrorFormatFileData                       = "%w: file data %s"
	indexErrorFormatExtendedAttribute              = "%w: extended attribute %q of inode %d"
	indexErrorFormatExtendedAttributeName          = "%w: extended attribute name %q"
	indexErrorFormatExtendedAttributeValueTooLarge = "%w: extended attribute value too large (%d bytes)"
	indexErrorFormatName                           = "%w: name %q"
	indexErrorFormatMoveDirectoryIntoItself        = "%w: cannot move a directory into itself"
	indexErrorFormatHashLength                     = "%w: hash of file data %s is %d bytes"
)

const (
	fileDataStateNameSucceeded        = "succeeded"
	fileDataStateNameLogicallyDeleted = "logically_deleted"
	fileDataStateNameUnknownFormat    = "FileDataState(%d)" // show unknown values as numbers
)

const (
	// fileDataHashLength is the byte length of FileData.Hash (SHA-256 of the file data bytes).
	fileDataHashLength = sha256.Size
	// rootDirectoryPermission is the permission of the root directory created at initialization.
	rootDirectoryPermission uint32 = 0o755
	// rootName is the name of the root directory, which has no entry in any directory.
	rootName = ""
	// unattachedInodeNumber means the file data is not attached to any inode (e.g. after logical deletion).
	unattachedInodeNumber = 0
	// noFileDataID means the inode has no file data (a directory).
	noFileDataID = ""
	// emptyName means no name was given. Not allowed as an entry or extended attribute name.
	emptyName = ""
	// nameTerminatorCharacter is the C string terminator, which names cannot contain.
	nameTerminatorCharacter = '\x00'
	// currentDirectoryName and parentDirectoryName have special meaning and cannot be entry names.
	currentDirectoryName = "."
	parentDirectoryName  = ".."
	// forbiddenNameCharacters are characters entry names cannot contain (path separator and C string terminator).
	forbiddenNameCharacters = "/\x00"
)

// indexSchema is created at the first start; the root directory has no entry, so it is outside the entry uniqueness.
var indexSchema = []string{
	`CREATE TABLE IF NOT EXISTS meta (
		key   TEXT PRIMARY KEY,
		value INTEGER NOT NULL
	) WITHOUT ROWID`,
	// AUTOINCREMENT keeps inode numbers from being reused after the highest one is removed
	`CREATE TABLE IF NOT EXISTS inode (
		inode_number        INTEGER PRIMARY KEY AUTOINCREMENT,
		parent_inode_number INTEGER NOT NULL,
		name                TEXT    NOT NULL,
		mode                INTEGER NOT NULL,
		uid                 INTEGER NOT NULL,
		gid                 INTEGER NOT NULL,
		size                INTEGER NOT NULL,
		access_time         INTEGER NOT NULL,
		modify_time         INTEGER NOT NULL,
		change_time         INTEGER NOT NULL,
		file_data_id        TEXT    NOT NULL
	)`,
	fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS inode_directory_entry ON inode (parent_inode_number, name)
		WHERE inode_number != %d`, RootInodeNumber),
	`CREATE TABLE IF NOT EXISTS extended_attribute (
		inode_number INTEGER NOT NULL,
		name         TEXT    NOT NULL,
		value        BLOB    NOT NULL,
		valid_utf8   INTEGER NOT NULL,
		PRIMARY KEY (inode_number, name)
	) WITHOUT ROWID`,
	`CREATE TABLE IF NOT EXISTS file_data (
		file_data_id         TEXT    PRIMARY KEY,
		inode_number         INTEGER NOT NULL,
		size                 INTEGER NOT NULL,
		hash                 BLOB    NOT NULL,
		state                INTEGER NOT NULL,
		logically_deleted_at INTEGER NOT NULL
	) WITHOUT ROWID`,
	`CREATE INDEX IF NOT EXISTS file_data_logically_deleted ON file_data (state, logically_deleted_at)`,
	objectsViewSchema,
}

const inodeColumns = `inode_number, parent_inode_number, name, mode, uid, gid, size,
	access_time, modify_time, change_time, file_data_id`

const fileDataColumns = `file_data_id, inode_number, size, hash, state, logically_deleted_at`

// Inode is a single index entry (file or directory).
type Inode struct {
	InodeNumber       uint64
	ParentInodeNumber uint64
	Name              string
	Mode              uint32 // ModeRegular / ModeDirectory | permission
	UID, GID          uint32
	Size              uint64
	AccessTime        int64
	ModifyTime        int64
	ChangeTime        int64
	FileDataID        string // empty for directories
	// nlink (always 1), rdev (always 0), blocks and blksize are derived, so not stored
}

func (i *Inode) IsDirectory() bool { return i.Mode&ModeTypeMask == ModeDirectory }

// FileDataState is the state of a file data.
type FileDataState uint8

const (
	FileDataSucceeded        FileDataState = 1
	FileDataLogicallyDeleted FileDataState = 3
)

func (s FileDataState) String() string {
	switch s {
	case FileDataSucceeded:
		return fileDataStateNameSucceeded
	case FileDataLogicallyDeleted:
		return fileDataStateNameLogicallyDeleted
	}
	return fmt.Sprintf(fileDataStateNameUnknownFormat, s)
}

// FileData is the file holding the bytes of one file. One-to-one with an inode.
type FileData struct {
	FileDataID  string
	InodeNumber uint64
	Size        uint64
	Hash        [fileDataHashLength]byte // SHA-256 of the file data bytes. YaagwaDB does not verify it
	State       FileDataState
	// LogicallyDeletedAt is when logical deletion was requested.
	LogicallyDeletedAt int64
}

// DirectoryEntry is one Readdir result.
type DirectoryEntry struct {
	Name  string
	Inode Inode
}

type scanner interface {
	Scan(destination ...any) error
}

func scanInode(row scanner) (*Inode, error) {
	inode := &Inode{}
	err := row.Scan(&inode.InodeNumber, &inode.ParentInodeNumber, &inode.Name, &inode.Mode, &inode.UID, &inode.GID,
		&inode.Size, &inode.AccessTime, &inode.ModifyTime, &inode.ChangeTime, &inode.FileDataID)
	return inode, err
}

func scanFileData(row scanner) (*FileData, error) {
	fileData := &FileData{}
	var hash []byte
	err := row.Scan(&fileData.FileDataID, &fileData.InodeNumber, &fileData.Size, &hash, &fileData.State, &fileData.LogicallyDeletedAt)
	if err != nil {
		return nil, err
	}
	if len(hash) != fileDataHashLength {
		return nil, fmt.Errorf(indexErrorFormatHashLength, errors.New(Corrupted), fileData.FileDataID, len(hash))
	}
	copy(fileData.Hash[:], hash)
	return fileData, nil
}

func initializeIndex(index *database) error {
	return index.update(func(transaction *sql.Tx) error {
		if err := execAll(transaction, indexSchema); err != nil {
			return err
		}
		var version uint64
		err := transaction.QueryRow(`SELECT value FROM meta WHERE key = ?`, metaKeySchemaVersion).Scan(&version)
		if err == nil {
			if version != schemaVersion {
				return fmt.Errorf(indexErrorFormatSchemaVersion, errors.New(NotSupported), version)
			}
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if err := putMeta(transaction, metaKeySchemaVersion, schemaVersion); err != nil {
			return err
		}
		now := time.Now().UnixNano()
		_, err = transaction.Exec(`INSERT INTO inode (`+inodeColumns+`) VALUES (?, ?, ?, ?, 0, 0, 0, ?, ?, ?, ?)`,
			RootInodeNumber, RootInodeNumber, rootName, ModeDirectory|rootDirectoryPermission, now, now, now, noFileDataID)
		return err
	})
}

func getMeta(transaction querier, key string) (int64, error) {
	var value int64
	err := transaction.QueryRow(`SELECT value FROM meta WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return value, err
}

func putMeta(transaction querier, key string, value int64) error {
	_, err := transaction.Exec(`INSERT INTO meta (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func getInode(transaction querier, inodeNumber uint64) (*Inode, error) {
	inode, err := scanInode(transaction.QueryRow(`SELECT `+inodeColumns+` FROM inode WHERE inode_number = ?`, inodeNumber))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(errorFormatInodeNumber, errors.New(NotFound), inodeNumber)
	}
	return inode, err
}

func insertInode(transaction querier, inode *Inode) error {
	result, err := transaction.Exec(`INSERT INTO inode (parent_inode_number, name, mode, uid, gid, size,
		access_time, modify_time, change_time, file_data_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inode.ParentInodeNumber, inode.Name, inode.Mode, inode.UID, inode.GID, inode.Size,
		inode.AccessTime, inode.ModifyTime, inode.ChangeTime, inode.FileDataID)
	if err != nil {
		return err
	}
	inodeNumber, err := result.LastInsertId()
	inode.InodeNumber = uint64(inodeNumber)
	return err
}

func updateInode(transaction querier, inode *Inode) error {
	_, err := transaction.Exec(`UPDATE inode SET parent_inode_number = ?, name = ?, mode = ?, uid = ?, gid = ?, size = ?,
		access_time = ?, modify_time = ?, change_time = ?, file_data_id = ? WHERE inode_number = ?`,
		inode.ParentInodeNumber, inode.Name, inode.Mode, inode.UID, inode.GID, inode.Size,
		inode.AccessTime, inode.ModifyTime, inode.ChangeTime, inode.FileDataID, inode.InodeNumber)
	return err
}

func deleteInode(transaction querier, inodeNumber uint64) error {
	_, err := transaction.Exec(`DELETE FROM inode WHERE inode_number = ?`, inodeNumber)
	return err
}

func lookupDirectoryEntry(transaction querier, parent uint64, name string) (*Inode, error) {
	inode, err := scanInode(transaction.QueryRow(`SELECT `+inodeColumns+` FROM inode
		WHERE parent_inode_number = ? AND name = ? AND inode_number != ?`, parent, name, RootInodeNumber))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return inode, err
}

func hasChildren(transaction querier, parent uint64) (bool, error) {
	var exists bool
	err := transaction.QueryRow(`SELECT EXISTS (SELECT 1 FROM inode WHERE parent_inode_number = ? AND inode_number != ?)`,
		parent, RootInodeNumber).Scan(&exists)
	return exists, err
}

func listDirectory(transaction querier, parent uint64) ([]DirectoryEntry, error) {
	rows, err := transaction.Query(`SELECT `+inodeColumns+` FROM inode
		WHERE parent_inode_number = ? AND inode_number != ? ORDER BY name`, parent, RootInodeNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []DirectoryEntry
	for rows.Next() {
		inode, err := scanInode(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, DirectoryEntry{Name: inode.Name, Inode: *inode})
	}
	return entries, rows.Err()
}

func countInodes(transaction querier) (uint64, error) {
	var count uint64
	err := transaction.QueryRow(`SELECT COUNT(*) FROM inode`).Scan(&count)
	return count, err
}

func checkParentDirectory(transaction querier, parent uint64) error {
	inode, err := getInode(transaction, parent)
	if err != nil {
		return err
	}
	if !inode.IsDirectory() {
		return fmt.Errorf(errorFormatInodeNumber, errors.New(NotDirectory), parent)
	}
	return nil
}

func getFileData(transaction querier, fileDataID string) (*FileData, error) {
	fileData, err := scanFileData(transaction.QueryRow(`SELECT `+fileDataColumns+` FROM file_data WHERE file_data_id = ?`, fileDataID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(indexErrorFormatFileData, errors.New(NotFound), fileDataID)
	}
	return fileData, err
}

func putFileData(transaction querier, fileData *FileData) error {
	_, err := transaction.Exec(`INSERT INTO file_data (`+fileDataColumns+`) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (file_data_id) DO UPDATE SET inode_number = excluded.inode_number, size = excluded.size,
		hash = excluded.hash, state = excluded.state, logically_deleted_at = excluded.logically_deleted_at`,
		fileData.FileDataID, fileData.InodeNumber, fileData.Size, fileData.Hash[:], fileData.State, fileData.LogicallyDeletedAt)
	return err
}

func deleteFileData(transaction querier, fileDataID string) error {
	_, err := transaction.Exec(`DELETE FROM file_data WHERE file_data_id = ?`, fileDataID)
	return err
}

func hasFileData(transaction querier, fileDataID string) (bool, error) {
	var exists bool
	err := transaction.QueryRow(`SELECT EXISTS (SELECT 1 FROM file_data WHERE file_data_id = ?)`, fileDataID).Scan(&exists)
	return exists, err
}

func listFileDataIDs(transaction querier) (map[string]bool, error) {
	rows, err := transaction.Query(`SELECT file_data_id FROM file_data`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	identifiers := map[string]bool{}
	for rows.Next() {
		var fileDataID string
		if err := rows.Scan(&fileDataID); err != nil {
			return nil, err
		}
		identifiers[fileDataID] = true
	}
	return identifiers, rows.Err()
}

func listExpiredFileData(transaction querier, threshold int64) ([]string, error) {
	rows, err := transaction.Query(`SELECT file_data_id FROM file_data WHERE state = ? AND logically_deleted_at <= ?`,
		FileDataLogicallyDeleted, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var expired []string
	for rows.Next() {
		var fileDataID string
		if err := rows.Scan(&fileDataID); err != nil {
			return nil, err
		}
		expired = append(expired, fileDataID)
	}
	return expired, rows.Err()
}

func countFileData(transaction querier, state FileDataState) (uint64, error) {
	var count uint64
	err := transaction.QueryRow(`SELECT COUNT(*) FROM file_data WHERE state = ?`, state).Scan(&count)
	return count, err
}

func getExtendedAttribute(transaction querier, inodeNumber uint64, name string) ([]byte, error) {
	var value []byte
	err := transaction.QueryRow(`SELECT value FROM extended_attribute WHERE inode_number = ? AND name = ?`, inodeNumber, name).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf(indexErrorFormatExtendedAttribute, errors.New(NotFound), name, inodeNumber)
	}
	return value, err
}

func putExtendedAttribute(transaction querier, inodeNumber uint64, name string, value []byte) error {
	_, err := transaction.Exec(`INSERT INTO extended_attribute (inode_number, name, value, valid_utf8) VALUES (?, ?, ?, ?)
		ON CONFLICT (inode_number, name) DO UPDATE SET value = excluded.value, valid_utf8 = excluded.valid_utf8`,
		inodeNumber, name, value, utf8.Valid(value))
	return err
}

func deleteExtendedAttribute(transaction querier, inodeNumber uint64, name string) error {
	_, err := transaction.Exec(`DELETE FROM extended_attribute WHERE inode_number = ? AND name = ?`, inodeNumber, name)
	return err
}

func deleteExtendedAttributes(transaction querier, inodeNumber uint64) error {
	_, err := transaction.Exec(`DELETE FROM extended_attribute WHERE inode_number = ?`, inodeNumber)
	return err
}

func listExtendedAttributes(transaction querier, inodeNumber uint64) ([]string, error) {
	rows, err := transaction.Query(`SELECT name FROM extended_attribute WHERE inode_number = ? ORDER BY name`, inodeNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func validateExtendedAttribute(name string, value []byte) error {
	if name == emptyName || len(name) > maximumNameLength || strings.ContainsRune(name, nameTerminatorCharacter) {
		return fmt.Errorf(indexErrorFormatExtendedAttributeName, errors.New(InvalidArgument), name)
	}
	if len(value) > MaximumExtendedAttributeValueLength {
		return fmt.Errorf(indexErrorFormatExtendedAttributeValueTooLarge, errors.New(InvalidArgument), len(value))
	}
	return nil
}

func ValidateName(name string) error {
	if name == emptyName || name == currentDirectoryName || name == parentDirectoryName || len(name) > maximumNameLength || strings.ContainsAny(name, forbiddenNameCharacters) {
		return fmt.Errorf(indexErrorFormatName, errors.New(InvalidArgument), name)
	}
	return nil
}

func registerFileData(transaction querier, payload *fileDataRegistration) (*Inode, error) {
	if err := checkParentDirectory(transaction, payload.Parent); err != nil {
		return nil, err
	}
	now := time.Now().UnixNano()
	inode := &Inode{
		ParentInodeNumber: payload.Parent, Name: payload.Name,
		Mode: ModeRegular | payload.Mode&^ModeTypeMask, UID: payload.UID, GID: payload.GID,
		Size: payload.Size, AccessTime: now, ModifyTime: payload.ModifyTime, ChangeTime: now, FileDataID: payload.FileDataID,
	}
	existing, err := lookupDirectoryEntry(transaction, payload.Parent, payload.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if existing.IsDirectory() {
			return nil, fmt.Errorf(errorFormatQuotedName, errors.New(IsDirectory), payload.Name)
		}
		if err := markLogicallyDeleted(transaction, existing.FileDataID, now); err != nil {
			return nil, err
		}
		inode.InodeNumber = existing.InodeNumber
		err = updateInode(transaction, inode)
	} else {
		err = insertInode(transaction, inode)
	}
	if err != nil {
		return nil, err
	}
	fileData := &FileData{FileDataID: payload.FileDataID, InodeNumber: inode.InodeNumber, Size: payload.Size, Hash: payload.Hash, State: FileDataSucceeded}
	return inode, putFileData(transaction, fileData)
}

func markLogicallyDeleted(transaction querier, fileDataID string, now int64) error {
	if fileDataID == noFileDataID {
		return nil
	}
	_, err := transaction.Exec(`UPDATE file_data SET state = ?, logically_deleted_at = ?, inode_number = ? WHERE file_data_id = ?`,
		FileDataLogicallyDeleted, now, unattachedInodeNumber, fileDataID)
	return err
}

func removeInode(transaction querier, inode *Inode, now int64) error {
	if err := markLogicallyDeleted(transaction, inode.FileDataID, now); err != nil {
		return err
	}
	if err := deleteExtendedAttributes(transaction, inode.InodeNumber); err != nil {
		return err
	}
	return deleteInode(transaction, inode.InodeNumber)
}

func lookup(transaction querier, parent uint64, name string) (*Inode, error) {
	inode, err := lookupDirectoryEntry(transaction, parent, name)
	if err != nil {
		return nil, err
	}
	if inode == nil {
		return nil, fmt.Errorf(daemonErrorFormatNameInParent, errors.New(NotFound), name, parent)
	}
	return inode, nil
}

func logicalDelete(transaction querier, parent uint64, name string) error {
	inode, err := lookup(transaction, parent, name)
	if err != nil {
		return err
	}
	if inode.IsDirectory() {
		return fmt.Errorf(errorFormatQuotedName, errors.New(IsDirectory), name)
	}
	return removeInode(transaction, inode, time.Now().UnixNano())
}

func makeDirectory(transaction querier, payload *directoryCreation) (*Inode, error) {
	if err := checkParentDirectory(transaction, payload.Parent); err != nil {
		return nil, err
	}
	existing, err := lookupDirectoryEntry(transaction, payload.Parent, payload.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf(errorFormatQuotedName, errors.New(Exists), payload.Name)
	}
	now := time.Now().UnixNano()
	inode := &Inode{
		ParentInodeNumber: payload.Parent, Name: payload.Name,
		Mode: ModeDirectory | payload.Mode&^ModeTypeMask, UID: payload.UID, GID: payload.GID,
		AccessTime: now, ModifyTime: now, ChangeTime: now, FileDataID: noFileDataID,
	}
	return inode, insertInode(transaction, inode)
}

func removeDirectory(transaction querier, parent uint64, name string) error {
	inode, err := lookup(transaction, parent, name)
	if err != nil {
		return err
	}
	if !inode.IsDirectory() {
		return fmt.Errorf(errorFormatQuotedName, errors.New(NotDirectory), name)
	}
	children, err := hasChildren(transaction, inode.InodeNumber)
	if err != nil {
		return err
	}
	if children {
		return fmt.Errorf(errorFormatQuotedName, errors.New(NotEmpty), name)
	}
	return removeInode(transaction, inode, time.Now().UnixNano())
}

func validateRename(transaction querier, payload *renameTarget, source *Inode) (*Inode, error) {
	if err := checkParentDirectory(transaction, payload.NewParent); err != nil {
		return nil, err
	}
	if source.IsDirectory() {
		// cannot move under itself or its descendants
		for ancestor := payload.NewParent; ; {
			if ancestor == source.InodeNumber {
				return nil, fmt.Errorf(indexErrorFormatMoveDirectoryIntoItself, errors.New(InvalidArgument))
			}
			if ancestor == RootInodeNumber {
				break
			}
			inode, err := getInode(transaction, ancestor)
			if err != nil {
				return nil, err
			}
			ancestor = inode.ParentInodeNumber
		}
	}
	target, err := lookupDirectoryEntry(transaction, payload.NewParent, payload.NewName)
	if err != nil || target == nil || target.InodeNumber == source.InodeNumber {
		return nil, err
	}
	targetHasChildren, err := hasChildren(transaction, target.InodeNumber)
	if err != nil {
		return nil, err
	}
	switch {
	case target.IsDirectory() && !source.IsDirectory():
		return nil, fmt.Errorf(errorFormatQuotedName, errors.New(IsDirectory), payload.NewName)
	case !target.IsDirectory() && source.IsDirectory():
		return nil, fmt.Errorf(errorFormatQuotedName, errors.New(NotDirectory), payload.NewName)
	case target.IsDirectory() && targetHasChildren:
		return nil, fmt.Errorf(errorFormatQuotedName, errors.New(NotEmpty), payload.NewName)
	}
	return target, nil
}

func rename(transaction querier, payload *renameTarget) error {
	source, err := lookup(transaction, payload.OldParent, payload.OldName)
	if err != nil {
		return err
	}
	target, err := validateRename(transaction, payload, source)
	if err != nil {
		return err
	}
	now := time.Now().UnixNano()
	if target != nil {
		if err := removeInode(transaction, target, now); err != nil {
			return err
		}
	}
	source.ParentInodeNumber = payload.NewParent
	source.Name = payload.NewName
	source.ChangeTime = now
	return updateInode(transaction, source)
}

func setAttribute(transaction querier, payload *attributeChange) (*Inode, error) {
	inode, err := getInode(transaction, payload.InodeNumber)
	if err != nil {
		return nil, err
	}
	attribute := payload.Attribute
	if payload.Mask&AttributeMode != noAttributeChange {
		inode.Mode = inode.Mode&ModeTypeMask | attribute.Mode&^ModeTypeMask
	}
	if payload.Mask&AttributeUID != noAttributeChange {
		inode.UID = attribute.UID
	}
	if payload.Mask&AttributeGID != noAttributeChange {
		inode.GID = attribute.GID
	}
	if payload.Mask&AttributeAccessTime != noAttributeChange {
		inode.AccessTime = attribute.AccessTime
	}
	if payload.Mask&AttributeModifyTime != noAttributeChange {
		inode.ModifyTime = attribute.ModifyTime
	}
	inode.ChangeTime = time.Now().UnixNano()
	return inode, updateInode(transaction, inode)
}
