//go:build linux || darwin || freebsd || openbsd || netbsd || dragonfly

package walker

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestNamedPipeWithSourceExtensionDoesNotBlock(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "input.go"), 0o600); err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		result Result
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		r, err := Scan(root, DefaultLimits())
		done <- outcome{r, err}
	}()
	select {
	case got := <-done:
		if got.err != nil || !got.result.Complete || len(got.result.Files) != 0 || len(got.result.Skipped) != 1 || got.result.Skipped[0].Reason != "non_regular" {
			t.Fatalf("unexpected FIFO result: %+v, %v", got.result, got.err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("scan blocked reading a FIFO")
	}
}

// This tests the regular-file-to-FIFO race independently of static Lstat
// filtering: the open is nonblocking and the changed type is rejected.
func TestReplacedFileCannotBlockAtOpen(t *testing.T) {
	root := t.TempDir()
	write(t, root, "input.go", "package main")
	before, err := os.Lstat(filepath.Join(root, "input.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "input.go")); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mkfifo(filepath.Join(root, "input.go"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	s := scanner{root: r, limits: DefaultLimits(), result: emptyResult()}
	if err := s.readFile("input.go", "go_source", before); err == nil {
		t.Fatal("a replaced nonregular file was accepted")
	}
}
