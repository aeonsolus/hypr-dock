package ctl

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestServeAndClientRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	stop, err := Serve(func(req Request) Response {
		if req.Cmd == CmdPing {
			return Response{Ok: true}
		}
		return Response{Ok: false, Error: "unknown: " + req.Cmd}
	})
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}

	if _, err := os.Stat(SocketPath()); err != nil {
		t.Fatalf("socket not created: %v", err)
	}

	resp, err := Send(Request{Cmd: CmdPing})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !resp.Ok {
		t.Fatalf("ping not ok: %+v", resp)
	}

	resp, err = Send(Request{Cmd: "nope"})
	if err != nil || resp.Ok || resp.Error == "" {
		t.Fatalf("unknown cmd handling: %+v %v", resp, err)
	}

	stop()

	// Socket removed on stop.
	if _, err := os.Stat(SocketPath()); !os.IsNotExist(err) {
		t.Fatal("socket not removed after stop")
	}
}

func TestServeReplacesStaleSocket(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	// Fake a crashed owner: bind a socket that stays behind on Close (like a
	// SIGKILLed process would).
	stale := filepath.Join(dir, "hypr-dock", "ctl.sock")
	os.MkdirAll(filepath.Dir(stale), 0700)
	owner, err := net.ListenUnix("unix", &net.UnixAddr{Name: stale, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	owner.SetUnlinkOnClose(false)
	owner.Close()

	// The bind file remains after Close; a new Serve must replace it.
	if _, err := os.Stat(stale); err != nil {
		t.Fatalf("stale socket setup failed: %v", err)
	}

	stop, err := Serve(func(Request) Response { return Response{Ok: true} })
	if err != nil {
		t.Fatalf("Serve over stale socket: %v", err)
	}
	defer stop()
}

func TestSendReportsMissingDock(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", dir)

	_, err := Send(Request{Cmd: CmdPing})
	if err == nil {
		t.Fatal("Send must fail when no dock listens")
	}
}
