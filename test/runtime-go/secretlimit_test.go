//go:build linux && (amd64 || arm64)

package hive

import (
	"syscall"
	"testing"
)

const rlimitMemlock = 8

func TestPastTheLockLimitHideAnswersAnError(t *testing.T) {
	if syscall.Geteuid() == 0 {
		t.Skip("root locks memory past any limit")
	}
	var was syscall.Rlimit
	if err := syscall.Getrlimit(rlimitMemlock, &was); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Setrlimit(rlimitMemlock, &syscall.Rlimit{Cur: 0, Max: was.Max}); err != nil {
		t.Fatal(err)
	}
	defer syscall.Setrlimit(rlimitMemlock, &was)
	got := Hide(string(make([]byte, 4*secretArenaSize)))
	if !got.IsError() || got.Err().Reason != "LimitExceeded" {
		t.Fatalf("hiding past the limit answered %v", got)
	}
	if Reveal(Bypass("still")) != "still" {
		t.Fatalf("bypass needed locked memory")
	}
}
