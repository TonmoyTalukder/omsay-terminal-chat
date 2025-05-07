package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: omsay-updater.exe <newFile> <targetExe>")
		return
	}

	newFile := os.Args[1]
	targetExe := os.Args[2]

	// Wait for original .exe to fully release (e.g., from Windows file lock)
	for i := 0; i < 20; i++ {
		err := os.Remove(targetExe)
		if err == nil {
			break
		}
		fmt.Println("⏳ Waiting to release lock on", targetExe)
		time.Sleep(1 * time.Second)
	}

	// Rename the new version to replace the target
	err := os.Rename(newFile, targetExe)
	if err != nil {
		fmt.Println("❌ Failed to replace:", err)
		return
	}

	// Restart the new version
	cmd := exec.Command(targetExe)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		fmt.Println("❌ Failed to restart:", err)
		return
	}

	fmt.Println("✅ OMSAY updated and restarted!")
}
