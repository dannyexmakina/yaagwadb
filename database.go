package yaagwadb

import (
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"modernc.org/sqlite"
)

const (
	sqliteDriverName = "sqlite"
	// queryDriverName is the driver of the connections that run read SQL. Its connections carry the authorizer.
	queryDriverName = "sqlite-query"
	// sqliteDataSourceFormat joins the file path and the query parameters of the data source name.
	sqliteDataSourceFormat = "file:%s?%s"
)

const (
	sqliteParameterPragma            = "_pragma"
	sqliteParameterTransactionLock   = "_txlock"
	sqliteTransactionLockImmediate   = "immediate"
	sqlitePragmaJournalModeWAL       = "journal_mode(WAL)"
	sqlitePragmaSynchronousFull      = "synchronous(FULL)"
	sqlitePragmaQueryOnly            = "query_only(1)"
	sqlitePragmaBusyTimeoutFormat    = "busy_timeout(%d)"
	sqlitePragmaForeignKeysOff       = "foreign_keys(0)"
	sqliteWriteConnectionCount       = 1
	sqliteUnlimitedReadConnectionAge = 0
)

// database is one SQLite file opened as one write connection and a pool of read connections.
type database struct {
	write *sql.DB
	read  *sql.DB
}

// querier is satisfied by *sql.Tx, so the same function reads in a read transaction and in a write transaction.
type querier interface {
	Exec(query string, arguments ...any) (sql.Result, error)
	Query(query string, arguments ...any) (*sql.Rows, error)
	QueryRow(query string, arguments ...any) *sql.Row
}

func dataSourceName(path string, busyTimeout time.Duration, parameters url.Values) string {
	parameters.Add(sqliteParameterPragma, sqlitePragmaJournalModeWAL)
	parameters.Add(sqliteParameterPragma, sqlitePragmaSynchronousFull)
	parameters.Add(sqliteParameterPragma, sqlitePragmaForeignKeysOff)
	parameters.Add(sqliteParameterPragma, fmt.Sprintf(sqlitePragmaBusyTimeoutFormat, busyTimeout.Milliseconds()))
	return fmt.Sprintf(sqliteDataSourceFormat, path, parameters.Encode())
}

func openDatabase(path string, busyTimeout time.Duration) (*database, error) {
	write, err := sql.Open(sqliteDriverName, dataSourceName(path, busyTimeout,
		url.Values{sqliteParameterTransactionLock: {sqliteTransactionLockImmediate}}))
	if err != nil {
		return nil, err
	}
	// One write connection runs the write transactions one at a time
	write.SetMaxOpenConns(sqliteWriteConnectionCount)
	write.SetConnMaxLifetime(sqliteUnlimitedReadConnectionAge)
	if err := write.Ping(); err != nil {
		write.Close()
		return nil, err
	}
	read, err := sql.Open(sqliteDriverName, dataSourceName(path, busyTimeout, url.Values{}))
	if err != nil {
		write.Close()
		return nil, err
	}
	return &database{write: write, read: read}, nil
}

func (d *database) close() error {
	if d == nil {
		return nil
	}
	readError := d.read.Close()
	if err := d.write.Close(); err != nil {
		return err
	}
	return readError
}

func (d *database) update(write func(transaction *sql.Tx) error) error {
	transaction, err := d.write.Begin()
	if err != nil {
		return err
	}
	if err := write(transaction); err != nil {
		_ = transaction.Rollback()
		return err
	}
	return transaction.Commit()
}

func (d *database) view(read func(transaction *sql.Tx) error) error {
	transaction, err := d.read.Begin()
	if err != nil {
		return err
	}
	defer transaction.Rollback()
	return read(transaction)
}

func execAll(transaction querier, statements []string) error {
	for _, statement := range statements {
		if _, err := transaction.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	sql.Register(queryDriverName, newQueryDriver())
}

func init() {
	// The objects view calls the function, so both drivers need it: the write connection creates the view
	sqlite.MustRegisterDeterministicScalarFunction(base64FunctionName, base64FunctionArgumentCount, encodeBase64Function)
}

func newQueryDriver() *sqlite.Driver {
	driver := &sqlite.Driver{}
	driver.MustRegisterDeterministicScalarFunction(base64FunctionName, base64FunctionArgumentCount, encodeBase64Function)
	driver.RegisterConnectionHook(installQueryAuthorizer)
	return driver
}
