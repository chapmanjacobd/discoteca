package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Connect opens a SQLite database and applies performance tuning PRAGMAs
// Slow query logging (50ms threshold) is enabled when SetDebugMode(true) is called
func Connect(dbPath string) (*sql.DB, error) {
	// Add busy timeout and immediate locking to handle concurrent writes better
	dsn := fmt.Sprintf("%s?_busy_timeout=30000&_txlock=immediate", dbPath)

	// Open the base database
	baseDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	// Apply performance tuning
	tuning := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-256000",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA foreign_keys=ON",
		"PRAGMA mmap_size=2147483648",
	}

	for _, pragma := range tuning {
		if _, err := baseDB.Exec(pragma); err != nil {
			baseDB.Close()
			return nil, fmt.Errorf("failed to apply pragma %q: %w", pragma, err)
		}
	}

	// Wrap with tracing connector if debug mode is enabled
	if IsDebugMode() {
		return wrapWithTracing(baseDB, dsn)
	}

	return baseDB, nil
}

// ConnectWithInit connects to a database and initializes it if needed
// Returns the database connection and a Queries object ready to use
func ConnectWithInit(dbPath string) (*sql.DB, *Queries, error) {
	sqlDB, err := Connect(dbPath)
	if err != nil {
		return nil, nil, err
	}

	if err := InitDB(sqlDB); err != nil {
		sqlDB.Close()
		return nil, nil, fmt.Errorf("failed to initialize database %s: %w", dbPath, err)
	}

	return sqlDB, New(sqlDB), nil
}

// wrapWithTracing wraps the database connection with query tracing
func wrapWithTracing(baseDB *sql.DB, dsn string) (*sql.DB, error) {
	// Get the underlying driver
	drv := baseDB.Driver()

	// Create a connector from the existing connection
	// We need to close the baseDB and reopen with tracing
	baseDB.Close()

	connector, err := driverConnector(drv, dsn)
	if err != nil {
		return nil, err
	}

	// Wrap the connector with tracing
	tracedConnector := &traceConnector{connector: connector}

	// Open database with traced connector
	return sql.OpenDB(tracedConnector), nil
}

func driverConnector(drv driver.Driver, dsn string) (driver.Connector, error) {
	// Check if driver implements DriverContext
	if driverCtx, ok := drv.(driver.DriverContext); ok {
		return driverCtx.OpenConnector(dsn)
	}
	// Fallback: create a simple connector
	return &simpleConnector{driver: drv, dsn: dsn}, nil
}

type simpleConnector struct {
	driver driver.Driver
	dsn    string
}

func (c *simpleConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return c.driver.Open(c.dsn)
}

func (c *simpleConnector) Driver() driver.Driver {
	return c.driver
}

// traceConnector wraps a driver.Connector to trace queries
type traceConnector struct {
	connector driver.Connector
}

func (t *traceConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := t.connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &traceConn{Conn: conn}, nil
}

func (t *traceConnector) Driver() driver.Driver {
	return t.connector.Driver()
}

// traceConn wraps a driver.Conn to trace query execution
type traceConn struct {
	driver.Conn
}

func (c *traceConn) Prepare(query string) (driver.Stmt, error) {
	stmt, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &traceStmt{Stmt: stmt, query: query}, nil
}

func (c *traceConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if connCtx, ok := c.Conn.(driver.ConnPrepareContext); ok {
		stmt, err := connCtx.PrepareContext(ctx, query)
		if err != nil {
			return nil, err
		}
		return &traceStmt{Stmt: stmt, query: query}, nil
	}
	return c.Prepare(query)
}

func (c *traceConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if execConn, ok := c.Conn.(driver.ExecerContext); ok {
		start := time.Now()
		result, err := execConn.ExecContext(ctx, query, args)
		logSlowQuery(query, args, start)
		return result, err
	}
	// Fallback for drivers that don't implement ExecerContext
	values := namedValuesToValues(args)
	start := time.Now()
	//lint:ignore SA1019 Fallback for backward compatibility with older drivers
	result, err := c.Conn.(driver.Execer).Exec(query, values)
	logSlowQuery(query, valuesToNamedValues(values), start)
	return result, err
}

func (c *traceConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if queryConn, ok := c.Conn.(driver.QueryerContext); ok {
		start := time.Now()
		rows, err := queryConn.QueryContext(ctx, query, args)
		logSlowQuery(query, args, start)
		return rows, err
	}
	// Fallback for drivers that don't implement QueryerContext
	values := namedValuesToValues(args)
	start := time.Now()
	//lint:ignore SA1019 Fallback for backward compatibility with older drivers
	rows, err := c.Conn.(driver.Queryer).Query(query, values)
	logSlowQuery(query, valuesToNamedValues(values), start)
	return rows, err
}

// traceStmt wraps a driver.Stmt to trace query execution
type traceStmt struct {
	driver.Stmt

	query string
}

func (s *traceStmt) Exec(args []driver.Value) (driver.Result, error) {
	start := time.Now()
	//lint:ignore SA1019 Fallback for backward compatibility with older drivers
	result, err := s.Stmt.Exec(args)
	logSlowQuery(s.query, valuesToNamedValues(args), start)
	return result, err
}

func (s *traceStmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if stmtCtx, ok := s.Stmt.(driver.StmtExecContext); ok {
		start := time.Now()
		result, err := stmtCtx.ExecContext(ctx, args)
		logSlowQuery(s.query, args, start)
		return result, err
	}
	// Fallback for drivers that don't implement StmtExecContext
	values := namedValuesToValues(args)
	return s.Exec(values)
}

func (s *traceStmt) Query(args []driver.Value) (driver.Rows, error) {
	start := time.Now()
	//lint:ignore SA1019 Fallback for backward compatibility with older drivers
	rows, err := s.Stmt.Query(args)
	if err != nil {
		return nil, err
	}
	logSlowQuery(s.query, valuesToNamedValues(args), start)
	return rows, err
}

func (s *traceStmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if stmtCtx, ok := s.Stmt.(driver.StmtQueryContext); ok {
		start := time.Now()
		rows, err := stmtCtx.QueryContext(ctx, args)
		logSlowQuery(s.query, args, start)
		return rows, err
	}
	// Fallback for drivers that don't implement StmtQueryContext
	values := namedValuesToValues(args)
	return s.Query(values)
}

func namedValuesToValues(args []driver.NamedValue) []driver.Value {
	values := make([]driver.Value, len(args))
	for i, arg := range args {
		values[i] = arg.Value
	}
	return values
}

func valuesToNamedValues(args []driver.Value) []driver.NamedValue {
	values := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		values[i] = driver.NamedValue{Ordinal: i, Value: arg}
	}
	return values
}

func logSlowQuery(query string, args []driver.NamedValue, startTime time.Time) {
	if !debugModeEnabled.Load() {
		return
	}

	duration := time.Since(startTime)
	if duration > SlowQueryThreshold {
		slog.Debug("slow query detected",
			"duration_ms", duration.Milliseconds(),
			"query", query,
			"args", args,
		)
	}
}
