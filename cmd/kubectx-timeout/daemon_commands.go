package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrf/kubectx-timeout/internal"
)

func cmdDaemonInstall() {
	// Detect the current binary path
	defaultBinaryPath := "/usr/local/bin/kubectx-timeout"
	if execPath, err := os.Executable(); err == nil {
		if absPath, err := filepath.Abs(execPath); err == nil {
			defaultBinaryPath = absPath
		}
	}

	// Create launchd manager
	manager, err := internal.NewLaunchdManager(defaultBinaryPath)
	if err != nil {
		log.Fatalf("Failed to create launchd manager: %v", err)
	}

	fmt.Println("Installing kubectx-timeout daemon with launchd")
	fmt.Printf("Binary path: %s\n", defaultBinaryPath)

	// Confirm
	fmt.Print("\nDo you want to proceed with the installation? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Installation cancelled")
		return
	}

	// Install
	if err := manager.Install(); err != nil {
		log.Fatalf("Failed to install daemon: %v", err)
	}

	fmt.Println("\n✓ Daemon plist installed successfully")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Start the daemon: kubectx-timeout daemon-start")
	fmt.Println("  2. Check status: kubectx-timeout daemon-status")
}

func cmdDaemonUninstall() {
	// Detect the current binary path
	defaultBinaryPath := "/usr/local/bin/kubectx-timeout"
	if execPath, err := os.Executable(); err == nil {
		if absPath, err := filepath.Abs(execPath); err == nil {
			defaultBinaryPath = absPath
		}
	}

	// Create launchd manager
	manager, err := internal.NewLaunchdManager(defaultBinaryPath)
	if err != nil {
		log.Fatalf("Failed to create launchd manager: %v", err)
	}

	fmt.Println("Uninstalling kubectx-timeout daemon from launchd")

	// Confirm
	fmt.Print("\nDo you want to proceed with the uninstallation? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Uninstallation cancelled")
		return
	}

	// Uninstall
	if err := manager.Uninstall(); err != nil {
		log.Fatalf("Failed to uninstall daemon: %v", err)
	}

	fmt.Println("\n✓ Daemon plist uninstalled successfully")
}

func cmdDaemonStart() {
	// Detect the current binary path
	defaultBinaryPath := "/usr/local/bin/kubectx-timeout"
	if execPath, err := os.Executable(); err == nil {
		if absPath, err := filepath.Abs(execPath); err == nil {
			defaultBinaryPath = absPath
		}
	}

	// Create launchd manager
	manager, err := internal.NewLaunchdManager(defaultBinaryPath)
	if err != nil {
		log.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Load daemon
	fmt.Println("Starting kubectx-timeout daemon...")
	if err := manager.Load(); err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	fmt.Println("✓ Daemon started successfully")
	fmt.Println("\nTo check status: kubectx-timeout daemon-status")
}

func cmdDaemonStop() {
	// Detect the current binary path
	defaultBinaryPath := "/usr/local/bin/kubectx-timeout"
	if execPath, err := os.Executable(); err == nil {
		if absPath, err := filepath.Abs(execPath); err == nil {
			defaultBinaryPath = absPath
		}
	}

	// Create launchd manager
	manager, err := internal.NewLaunchdManager(defaultBinaryPath)
	if err != nil {
		log.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Unload daemon
	fmt.Println("Stopping kubectx-timeout daemon...")
	if err := manager.Unload(); err != nil {
		log.Fatalf("Failed to stop daemon: %v", err)
	}

	fmt.Println("✓ Daemon stopped successfully")
}

func cmdDaemonRestart() {
	// Detect the current binary path
	defaultBinaryPath := "/usr/local/bin/kubectx-timeout"
	if execPath, err := os.Executable(); err == nil {
		if absPath, err := filepath.Abs(execPath); err == nil {
			defaultBinaryPath = absPath
		}
	}

	// Create launchd manager
	manager, err := internal.NewLaunchdManager(defaultBinaryPath)
	if err != nil {
		log.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Restart daemon
	fmt.Println("Restarting kubectx-timeout daemon...")
	if err := manager.Restart(); err != nil {
		log.Fatalf("Failed to restart daemon: %v", err)
	}

	fmt.Println("✓ Daemon restarted successfully")
	fmt.Println("\nTo check status: kubectx-timeout daemon-status")
}

func cmdDaemonStatus() {
	// Detect the current binary path
	defaultBinaryPath := "/usr/local/bin/kubectx-timeout"
	if execPath, err := os.Executable(); err == nil {
		if absPath, err := filepath.Abs(execPath); err == nil {
			defaultBinaryPath = absPath
		}
	}

	// Create launchd manager
	manager, err := internal.NewLaunchdManager(defaultBinaryPath)
	if err != nil {
		log.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Get status
	status, err := manager.GetStatus()
	if err != nil {
		log.Fatalf("Failed to get daemon status: %v", err)
	}

	fmt.Print(status)
}
