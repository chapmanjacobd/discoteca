package syncweb

import (
	"os"
	"testing"
	"time"
)

func TestNodeLifecycle(t *testing.T) {
	home, err := os.MkdirTemp("", "node-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(home)

	node, err := NewNode(home, "test-node", "tcp://127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	if node.IsRunning() {
		t.Error("node should not be running yet")
	}

	if err := node.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}

	if !node.IsRunning() {
		t.Error("node should be running")
	}

	// Double start should be a no-op
	if err := node.Start(); err != nil {
		t.Errorf("double start failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	node.Stop()
	if node.IsRunning() {
		t.Error("node should not be running after stop")
	}

	// Double stop should be a no-op
	node.Stop()
}

func TestNodeRestart(t *testing.T) {
	home, err := os.MkdirTemp("", "node-restart-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(home)

	node, err := NewNode(home, "restart-node", "tcp://127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	if err := node.Start(); err != nil {
		t.Fatalf("failed to start node: %v", err)
	}
	node.Stop()

	// Re-create node from same home
	node2, err := NewNode(home, "restart-node", "tcp://127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to re-create node: %v", err)
	}

	if err := node2.Start(); err != nil {
		t.Fatalf("failed to restart node: %v", err)
	}
	node2.Stop()
}

func TestNodeInvalidHome(t *testing.T) {
	// Try a read-only or non-existent path where we can't write
	_, err := NewNode("/root/should-not-be-writable", "test", "")
	if err == nil {
		t.Error("expected error with invalid home dir")
	}
}
