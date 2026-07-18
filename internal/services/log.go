package services

import (
	"fmt"
	"os"
	"time"
)

const maxLogBytes = 5 * 1024 * 1024 // 5MB
const maxOutputChars = 4000

func RotateLogIfLarge(logPath string) {
	info, err := os.Stat(logPath)
	if err != nil || info.Size() < maxLogBytes {
		return
	}
	_ = os.Rename(logPath, logPath+".1")
}

func appendLog(logPath, output string, exitCode int, duration time.Duration) {
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		log.Printf("zdai: open log file: %v", err)
		return
	}
	defer f.Close()

	entry := fmt.Sprintf(
		"=== %s exit=%d duration=%s ===\n%s\n\n",
		time.Now().Format(time.RFC3339),
		exitCode,
		duration.Round(time.Second),
		output,
	)
	if _, err := f.WriteString(entry); err != nil {
		log.Printf("zdai: write log file: %v", err)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "...(truncated)...\n" + s[len(s)-n:]
}
