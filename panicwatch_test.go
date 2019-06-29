package panicwatch

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// each test uses this test method to run a separate process in order to test the functionality
func helperProcess(args ...string) (*exec.Cmd, *bytes.Buffer, *bytes.Buffer) {
	cmd := exec.Command("./test", args...)
	cmd.Stderr = new(bytes.Buffer)
	cmd.Stdout = new(bytes.Buffer)

	return cmd, cmd.Stdout.(*bytes.Buffer), cmd.Stderr.(*bytes.Buffer)
}

func TestNoPanic(t *testing.T) {
	cmd, stdout, stderr := helperProcess("no-panic")
	if err := cmd.Run(); err != nil {
		t.Fatalf("unexpected error: %s\nstderr: %s", err, stderr.String())
	}

	if stdout.String() != "some stdout output\n" {
		t.Fatalf("unexpected stdout: %#v", stdout.String())
	}

	if stderr.String() != "some stderr output\n" {
		t.Fatalf("unexpected stderr: %#v", stderr.String())
	}
}

func TestNoPanicError(t *testing.T) {
	cmd, _, stderr := helperProcess("no-panic-error")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.(*exec.ExitError).ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got: %d\n", err.(*exec.ExitError).ExitCode())
	}

	if stderr.String() != "blah blah something happened\n" {
		t.Fatalf("unexpected stderr: %#v", stderr.String())
	}
}

func TestPanic(t *testing.T) {
	cmd, stdout, stderr := helperProcess("panic")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.(*exec.ExitError).ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got: %d\n", err.(*exec.ExitError).ExitCode())
	}

	if !strings.HasPrefix(stdout.String(), "some output...\neverything looks good...\n") {
		t.Fatalf("unexpected stdout: %#v", stdout.String())
	}

	if !strings.HasPrefix(stderr.String(), "panic: wtf, unexpected panic!\n\ngoroutine 1 [running]") {
		t.Fatalf("unexpected stderr: %#v", stderr.String())
	}

	if !strings.HasSuffix(stdout.String(), "caught panic: wtf, unexpected panic!\n") {
		t.Fatalf("failed to catch panic, stderr: %#v", stderr.String())
	}
}

func TestPanicAndError(t *testing.T) {
	cmd, stdout, stderr := helperProcess("panic-and-error")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.(*exec.ExitError).ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got: %d\n", err.(*exec.ExitError).ExitCode())
	}

	if !strings.HasPrefix(stdout.String(), "some output...\neverything looks good...\n") {
		t.Fatalf("unexpected stdout: %#v", stdout.String())
	}

	if !strings.HasPrefix(stderr.String(), "well something goes bad ...\npanic: ... and panic!") {
		t.Fatalf("unexpected stderr: %#v", stderr.String())
	}

	if !strings.HasSuffix(stdout.String(), "caught panic: ... and panic!\n") {
		t.Fatalf("failed to catch panic, stderr: %#v", stderr.String())
	}
}

func TestPanicSyncSplit(t *testing.T) {
	cmd, stdout, stderr := helperProcess("panic-sync-split")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.(*exec.ExitError).ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got: %d\n", err.(*exec.ExitError).ExitCode())
	}

	if stdout.String() != "caught panic: i'm split in two lol\n" {
		t.Fatalf("failed to catch panic, stderr: %#v", stderr.String())
	}

	if !strings.HasPrefix(stderr.String(), "panic: i'm split in two lol\n") {
		t.Fatalf("unexpected stderr: %#v", stderr.String())
	}
}

func TestPanicWithGarbage(t *testing.T) {
	cmd, stdout, stderr := helperProcess("panic-with-garbage")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.(*exec.ExitError).ExitCode() != 2 {
		t.Fatalf("expected exit code 2, got: %d\n", err.(*exec.ExitError).ExitCode())
	}

	if stdout.String() != "caught panic: and BAM!\n" {
		t.Fatalf("failed to catch panic, stderr: %#v", stderr.String())
	}

	if !strings.HasPrefix(stderr.String(), "panic: blah blah\n\nsome garbage here...\nsome garbage") {
		t.Fatalf("unexpected stderr: %#v", stderr.String())
	}
}
