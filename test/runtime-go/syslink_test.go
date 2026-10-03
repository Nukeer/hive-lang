package hive

import (
	"os"
	"path/filepath"
	"testing"
)

// **A key set by the program is the key from then on, and is kept nowhere else.**
// Not the environment, which another program on the machine would read, and not
// `~/.hive/syslink.key`, which every other program on it does.
func TestSetKeyIsUsedAndNeverSaved(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HIVE_SYSLINK_KEY", "from the environment")

	SyslinkSetKey(Bypass("first"))
	if got, err := clusterSecret(); err != nil || Reveal(got) != "first" {
		t.Fatalf("after setKey(first) the key is %q (%v)", Reveal(got), err)
	}
	SyslinkSetKey(Bypass("second"))
	if got, err := clusterSecret(); err != nil || Reveal(got) != "second" {
		t.Fatalf("after setKey(second) the key is %q (%v)", Reveal(got), err)
	}

	if _, err := os.Stat(filepath.Join(home, ".hive", "syslink.key")); !os.IsNotExist(err) {
		t.Fatalf("a key file was written: %v", err)
	}
	if os.Getenv("HIVE_SYSLINK_KEY") != "from the environment" {
		t.Fatalf("the environment was changed")
	}
}

func TestAnEmptyKeyIsRefused(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("setKey(\"\") was accepted")
		}
	}()
	SyslinkSetKey(Bypass(""))
}
