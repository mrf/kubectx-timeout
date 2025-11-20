package internal

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestPIDFile_AcquireAndRelease(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "daemon.pid")

	pidFile := &PIDFile{path: pidPath}

	// Test Acquire
	err := pidFile.Acquire()
	if err != nil {
		t.Fatalf("Failed to acquire PID file: %v", err)
	}

	// Verify PID file exists
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Error("PID file should exist after Acquire")
	}

	// Verify PID file contains current PID
	pid, err := pidFile.ReadPID()
	if err != nil {
		t.Fatalf("Failed to read PID: %v", err)
	}

	currentPID := os.Getpid()
	if pid != currentPID {
		t.Errorf("Expected PID %d, got %d", currentPID, pid)
	}

	// Test Release
	err = pidFile.Release()
	if err != nil {
		t.Fatalf("Failed to release PID file: %v", err)
	}

	// Verify PID file is removed
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Error("PID file should not exist after Release")
	}
}

func TestPIDFile_AcquireTwice(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "daemon.pid")

	pidFile1 := &PIDFile{path: pidPath}
	pidFile2 := &PIDFile{path: pidPath}

	// First acquire should succeed
	err := pidFile1.Acquire()
	if err != nil {
		t.Fatalf("First acquire failed: %v", err)
	}
	defer pidFile1.Release()

	// Second acquire should fail
	err = pidFile2.Acquire()
	if err == nil {
		t.Error("Second acquire should fail when process is running")
	}
}

func TestPIDFile_StalePIDFile(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "daemon.pid")

	// Write a stale PID (unlikely to be a real running process)
	stalePID := 999999
	err := os.WriteFile(pidPath, []byte(strconv.Itoa(stalePID)+"\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write stale PID file: %v", err)
	}

	pidFile := &PIDFile{path: pidPath}

	// Acquire should succeed by removing stale PID file
	err = pidFile.Acquire()
	if err != nil {
		t.Fatalf("Failed to acquire with stale PID file: %v", err)
	}
	defer pidFile.Release()

	// Verify new PID is current process
	pid, err := pidFile.ReadPID()
	if err != nil {
		t.Fatalf("Failed to read PID: %v", err)
	}

	if pid != os.Getpid() {
		t.Errorf("Expected current PID %d, got %d", os.Getpid(), pid)
	}
}

func TestPIDFile_GetPath(t *testing.T) {
	pidFile := NewPIDFile()
	path := pidFile.GetPath()

	if path == "" {
		t.Error("Expected non-empty PID file path")
	}

	stateDir := GetStateDir()
	expectedPath := filepath.Join(stateDir, "daemon.pid")

	if path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, path)
	}
}

func TestPIDFile_ReadPID_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "nonexistent.pid")

	pidFile := &PIDFile{path: pidPath}

	_, err := pidFile.ReadPID()
	if err == nil {
		t.Error("Expected error when reading non-existent PID file")
	}
}

func TestPIDFile_ReadPID_InvalidContent(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "invalid.pid")

	// Write invalid content
	err := os.WriteFile(pidPath, []byte("not-a-number\n"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid PID file: %v", err)
	}

	pidFile := &PIDFile{path: pidPath}

	_, err = pidFile.ReadPID()
	if err == nil {
		t.Error("Expected error when reading invalid PID file")
	}
}
