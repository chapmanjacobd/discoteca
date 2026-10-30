package server

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/chapmanjacobd/discoteca/internal/db"
	"github.com/chapmanjacobd/discoteca/internal/fileutil"
	"github.com/chapmanjacobd/discoteca/internal/logging"
)

// getOrCreateDBConn retrieves a cached database connection or creates a new one
func (c *Server) getOrCreateDBConn(ctx context.Context, dbPath string) (*sql.DB, error) {
	if val, ok := c.dbCache.Load(dbPath); ok {
		if dbConn, ok := val.(*sql.DB); ok {
			return dbConn, nil
		}
	}

	newDB, err := db.Connect(ctx, dbPath)
	if err != nil {
		return nil, err
	}

	sqlDB := newDB
	if loaded, ok := c.dbCache.LoadOrStore(dbPath, sqlDB); ok {
		if dbConn, ok := loaded.(*sql.DB); ok {
			sqlDB = dbConn
		}
		_ = newDB.Close()
	}
	return sqlDB, nil
}

// handleCorruptionError attempts to repair a corrupted database and returns whether to retry
func (c *Server) handleCorruptionError(
	ctx context.Context,
	dbPath string,
	sqlDB *sql.DB,
	err error,
	isConnectError bool,
) (shouldRetry bool, wrappedErr error) {
	if !db.IsCorruptionError(err) {
		return false, nil
	}

	if isConnectError {
		if repErr := db.Repair(ctx, dbPath); repErr != nil {
			return false, fmt.Errorf("repair failed: %w (original error: %w)", repErr, err)
		}
		logging.Log.Info("Database repaired, retrying connect", "db", dbPath)
		return true, nil
	}

	// Query error: delete from cache and close connection
	c.dbCache.Delete(dbPath)
	_ = sqlDB.Close()

	if repErr := db.Repair(ctx, dbPath); repErr != nil {
		logging.Log.Error("Database repair failed", "db", dbPath, "error", repErr)
		return false, nil // Return original error
	}
	logging.Log.Info("Database repaired, retrying operation", "db", dbPath)
	return true, nil
}

// execDB connects to the database and executes fn. If a corruption error occurs,
// it attempts to repair the database and retries the operation once.
func (c *Server) execDB(ctx context.Context, dbPath string, fn func(ctx context.Context, sqlDB *sql.DB) error) error {
	const maxRetries = 1
	for i := 0; i <= maxRetries; i++ {
		sqlDB, err := c.getOrCreateDBConn(ctx, dbPath)
		if err != nil {
			if shouldRetry, wrappedErr := c.handleCorruptionError(ctx, dbPath, nil, err, true); shouldRetry {
				continue
			} else if wrappedErr != nil {
				return wrappedErr
			}
			return err
		}

		err = fn(ctx, sqlDB)
		if err != nil {
			if shouldRetry, _ := c.handleCorruptionError(ctx, dbPath, sqlDB, err, false); shouldRetry {
				continue
			}
			if i > 0 {
				logging.Log.Error("Operation failed even after database repair", "db", dbPath, "error", err)
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("operation failed after %d retries", maxRetries)
}

// Close closes all cached database connections
func (c *Server) Close() error {
	var errs []error
	c.dbCache.Range(func(key, value any) bool {
		if sqlDB, ok := value.(*sql.DB); ok {
			if err := sqlDB.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		c.dbCache.Delete(key)
		return true
	})

	if len(errs) > 0 {
		return fmt.Errorf("failed to close some resources: %v", errs)
	}
	return nil
}

// setupAPIToken sets the API token from the environment or generates one. A
// generated token is printed once to stderr; it is never embedded in a static
// response.
func (c *Server) setupAPIToken() {
	if envToken := os.Getenv("DISCO_API_TOKEN"); envToken != "" {
		c.APIToken = envToken
		return
	}
	c.APIToken = fileutil.RandomString(32)
	fmt.Fprintf(
		os.Stderr,
		"DISCO_API_TOKEN is not set. Generated API token: %s\n"+
			"Use this token to authenticate (POST /api/session with an Authorization: Bearer header).\n",
		c.APIToken,
	)
}

// listenAddress returns the configured host and port as a net address.
func (c *Server) listenAddress() string {
	host := c.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, strconv.Itoa(c.Port))
}

// isLoopbackHost reports whether host refers to the local machine only.
func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	switch strings.ToLower(host) {
	case "", "localhost":
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// validateListenConfig refuses an unauthenticated bind on a non-loopback
// interface.
func (c *Server) validateListenConfig() error {
	if !isLoopbackHost(c.Host) && os.Getenv("DISCO_API_TOKEN") == "" {
		return fmt.Errorf(
			"refusing to listen on non-loopback address %q without DISCO_API_TOKEN; "+
				"set DISCO_API_TOKEN or bind to 127.0.0.1 and use a reverse proxy",
			c.Host,
		)
	}
	return nil
}

// cacheDatabases connects to all databases and stores them in the cache
func (c *Server) cacheDatabases(ctx context.Context) {
	for _, dbPath := range c.Databases {
		sqlDB, _, err := db.ConnectWithInit(ctx, dbPath)
		if err != nil {
			logging.Log.Error("Failed to connect to database on startup", "db", dbPath, "error", err)
			continue
		}
		c.dbCache.Store(dbPath, sqlDB)
	}
}

// runMaintenance runs maintenance on all databases asynchronously
func (c *Server) runMaintenance(ctx context.Context) {
	go func() {
		config := db.DefaultMaintenanceConfig()
		for _, dbPath := range c.Databases {
			if val, ok := c.dbCache.Load(dbPath); ok {
				if sqlDB, ok := val.(*sql.DB); ok {
					if err := db.RunMaintenance(ctx, sqlDB, config, dbPath); err != nil {
						logging.Log.Error("Maintenance failed", "db", dbPath, "error", err)
					}
				}
			}
		}
	}()
}

// checkFfmpeg checks if ffmpeg is available in PATH
func (c *Server) checkFfmpeg() {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		logging.Log.Warn("ffmpeg not found in PATH, on-the-fly transcoding will be unavailable")
		c.hasFfmpeg = false
	} else {
		c.hasFfmpeg = true
	}
}

// openBrowser opens the default browser to the server URL
func (c *Server) openBrowser(baseURL string) {
	go func() {
		time.Sleep(500 * time.Millisecond)
		var openCmd string
		var openArgs []string

		switch runtime.GOOS {
		case "linux":
			openCmd = "xdg-open"
		case "darwin":
			openCmd = "open"
		case "windows":
			openCmd = "cmd"
			openArgs = []string{"/c", "start"}
		}

		if openCmd != "" {
			openArgs = append(openArgs, baseURL)
			cmd := exec.Command(openCmd, openArgs...)
			if err := cmd.Start(); err != nil {
				logging.Log.Debug("Failed to open browser", "error", err)
				return
			}
			go func() { _ = cmd.Wait() }()
		}
	}()
}

// Run starts the HTTP server
func (c *Server) Run(ctx context.Context) error {
	defer c.Close()
	logging.SetupLogging(c.Verbose)
	db.InitFtsConfig()
	db.SetCacheSizeMB(c.DBCacheMB)

	if err := c.validateListenConfig(); err != nil {
		return err
	}
	if err := c.validateKiwixConfig(); err != nil {
		return err
	}

	// Initialize MIME types
	initMimeTypes()

	c.ApplicationStartTime = time.Now().UnixNano()

	// Setup API token
	c.setupAPIToken()

	// Connect and cache all databases
	c.cacheDatabases(ctx)

	// Run maintenance on all databases (refresh folder_stats and FTS if needed)
	// This runs asynchronously so it doesn't block server startup
	c.runMaintenance(ctx)

	// Check for ffmpeg
	c.checkFfmpeg()

	handler := c.Mux()

	addr := c.listenAddress()
	baseURL := fmt.Sprintf("http://%s", addr)
	logging.Log.Info("Server starting", "addr", baseURL)

	// Open browser unless --no-browser is passed
	if !c.NoBrowser {
		c.openBrowser(baseURL)
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // Streaming responses (HLS, Raw files) need no write timeout or a very large one
		IdleTimeout:  120 * time.Second,
	}

	return server.ListenAndServe()
}

// searchTimeout returns the configured per-search timeout.
func (c *Server) searchTimeout() time.Duration {
	if c.SearchTimeout <= 0 {
		return 15 * time.Second
	}
	return c.SearchTimeout
}

// acquireSearchSlot blocks until a search slot is free or ctx is done. The
// returned function must be called to release the slot. A limit of 0 disables
// the check.
func (c *Server) acquireSearchSlot(ctx context.Context) (func(), error) {
	if c.MaxConcurrentSearches <= 0 {
		return func() {}, nil
	}
	c.searchSemOnce.Do(func() {
		c.searchSem = make(chan struct{}, c.MaxConcurrentSearches)
	})
	select {
	case c.searchSem <- struct{}{}:
		return func() { <-c.searchSem }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// execWriteAcrossDBs runs fn against every configured database and returns a
// write outcome. Handlers use it so a failure is not reported as success.
func (c *Server) execWriteAcrossDBs(
	ctx context.Context,
	fn func(context.Context, *sql.DB) (int64, error),
) (affected int64, succeeded, failed int, errs []string) {
	for _, dbPath := range c.Databases {
		var rows int64
		err := c.execDB(ctx, dbPath, func(ctx context.Context, sqlDB *sql.DB) error {
			var err error
			rows, err = fn(ctx, sqlDB)
			return err
		})
		if err != nil {
			failed++
			errs = append(errs, fmt.Sprintf("%s: %v", dbPath, err))
			logging.Log.Error("Database write failed", "db", dbPath, "error", err)
			continue
		}
		succeeded++
		affected += rows
	}
	return affected, succeeded, failed, errs
}

// sendWriteResult maps a multi-database write outcome to an HTTP response.
// notFoundMessage, when set, turns a zero-row success into a 404.
func sendWriteResult(
	w http.ResponseWriter,
	successStatus int,
	affected int64,
	succeeded, failed int,
	errs []string,
	notFoundMessage string,
) {
	if failed == 0 {
		if affected == 0 && notFoundMessage != "" {
			sendError(w, http.StatusNotFound, notFoundMessage)
			return
		}
		w.WriteHeader(successStatus)
		return
	}
	if succeeded == 0 {
		sendJSON(w, http.StatusInternalServerError, map[string]any{
			"error":    "all database writes failed",
			"affected": affected,
			"failed":   failed,
			"errors":   errs,
		})
		return
	}
	sendJSON(w, http.StatusMultiStatus, map[string]any{
		"status":   "partial",
		"affected": affected,
		"failed":   failed,
		"errors":   errs,
	})
}
