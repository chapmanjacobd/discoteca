package syncweb

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/syncthing/syncthing/lib/config"
	"github.com/syncthing/syncthing/lib/events"
	"github.com/syncthing/syncthing/lib/protocol"
	"github.com/syncthing/syncthing/lib/svcutil"
	"github.com/syncthing/syncthing/lib/syncthing"
	"github.com/syncthing/syncthing/lib/tlsutil"
)

type Node struct {
	App      *syncthing.App
	Cfg      config.Wrapper
	EvLogger events.Logger
	Ctx      context.Context
	Cancel   context.CancelFunc
	db       io.Closer
	running  bool
	mu       sync.RWMutex
}

func NewNode(homeDir string, name string, listenAddr string) (*Node, error) {
	if homeDir == "" {
		var err error
		homeDir, err = os.MkdirTemp("", "disco-syncweb-")
		if err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(homeDir, 0o700); err != nil {
		return nil, err
	}

	setupLogging(homeDir)

	ctx, cancel := context.WithCancel(context.Background())

	// Set up event logger
	evLogger := events.NewLogger()
	go evLogger.Serve(ctx)

	// Construct paths manually to avoid global locations state
	certPath := filepath.Join(homeDir, "cert.pem")
	keyPath := filepath.Join(homeDir, "key.pem")
	cfgPath := filepath.Join(homeDir, "config.xml")
	dbPath := filepath.Join(homeDir, "index-v2")

	// Load or create certificate
	cert, err := tlsutil.NewCertificate(certPath, keyPath, "syncthing", 0, false)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	myID := protocol.NewDeviceID(cert.Certificate[0])

	// Load or create config
	var cfg config.Wrapper
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		slog.Info("Creating new Syncthing config", "path", cfgPath)
		newCfg := config.New(myID)
		// Customize defaults similar to syncweb-py
		newCfg.Options.StartBrowser = false
		newCfg.Options.URAccepted = -1 // Disable usage reporting
		newCfg.Options.LocalAnnEnabled = true
		newCfg.Options.GlobalAnnEnabled = true
		newCfg.GUI.Enabled = false

		if listenAddr != "" {
			newCfg.Options.RawListenAddresses = []string{listenAddr}
		} else {
			newCfg.Options.RawListenAddresses = []string{"tcp://127.0.0.1:0"}
		}

		cfg = config.Wrap(cfgPath, newCfg, myID, evLogger)
		go cfg.Serve(ctx)
		if err := cfg.Save(); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to save config: %w", err)
		}
	} else {
		slog.Info("Loading existing Syncthing config", "path", cfgPath)
		var err error
		cfg, _, err = config.Load(cfgPath, myID, evLogger)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		go cfg.Serve(ctx)
	}

	// Open database
	dbDeleteRetentionInterval := time.Duration(4320) * time.Hour
	sdb, err := syncthing.OpenDatabase(dbPath, dbDeleteRetentionInterval)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	appOpts := syncthing.Options{
		NoUpgrade:    true,
		ProfilerAddr: "",
	}

	app, err := syncthing.New(cfg, sdb, evLogger, cert, appOpts)
	if err != nil {
		sdb.Close()
		cancel()
		return nil, fmt.Errorf("failed to create Syncthing app: %w", err)
	}

	return &Node{
		App:      app,
		Cfg:      cfg,
		EvLogger: evLogger,
		Ctx:      ctx,
		Cancel:   cancel,
		db:       sdb,
	}, nil
}

func (n *Node) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.running {
		return nil
	}
	if err := n.App.Start(); err != nil {
		return err
	}
	n.running = true
	return nil
}

func (n *Node) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.running
}

func (n *Node) Serve() error {
	n.App.Wait()
	return nil
}

func (n *Node) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.running {
		return
	}
	n.App.Stop(svcutil.ExitSuccess)
	n.Cancel()
	n.App.Wait()
	n.running = false
	if n.db != nil {
		n.db.Close()
	}
}

func (n *Node) Close() error {
	n.Stop()
	return nil
}

func (n *Node) MyID() protocol.DeviceID {
	return n.Cfg.MyID()
}

func (n *Node) Subscribe(mask events.EventType) events.Subscription {
	return n.EvLogger.Subscribe(mask)
}
