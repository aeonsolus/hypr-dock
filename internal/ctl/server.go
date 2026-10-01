package ctl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

// Handler processes one request. It runs on the server's accept goroutine;
// anything that must touch GTK schedules itself via glib.IdleAdd.
type Handler func(req Request) Response

// Serve starts the control socket server. Stale sockets from a crashed
// previous owner are removed first. Returns an error only if the socket
// cannot be created; serving continues until Close is called.
func Serve(handler Handler) (func(), error) {
	dir := RuntimeDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("ctl: runtime dir: %w", err)
	}

	path := SocketPath()
	os.Remove(path) // stale socket from a dead owner

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("ctl: listen: %w", err)
	}

	var wg sync.WaitGroup
	closing := make(chan struct{})

	go func() {
		<-closing
		listener.Close()
	}()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-closing:
					return
				default:
					continue
				}
			}

			wg.Add(1)
			go func(conn net.Conn) {
				defer wg.Done()
				defer conn.Close()
				handleConn(conn, handler)
			}(conn)
		}
	}()

	if v := os.Getenv("HYPR_DOCK_CTL_DEBUG"); v == "1" {
		fmt.Fprintln(os.Stderr, "ctl: socket ready at", path)
	}

	stop := func() {
		select {
		case <-closing:
		default:
			close(closing)
		}
		os.Remove(path)
		wg.Wait()
	}

	return stop, nil
}

func handleConn(conn net.Conn, handler Handler) {
	conn.SetDeadline(deadline())

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return
	}

	var req Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		writeResponse(conn, Response{Ok: false, Error: "bad request: " + err.Error()})
		return
	}

	writeResponse(conn, handler(req))
}

func writeResponse(conn net.Conn, resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	conn.Write(append(data, '\n'))
}
