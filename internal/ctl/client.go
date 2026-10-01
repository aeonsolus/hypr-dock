package ctl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const requestTimeout = 2 * time.Second

func deadline() time.Time {
	return time.Now().Add(requestTimeout)
}

// Send sends one command to the running dock and returns its response.
// A missing dock yields a descriptive error rather than a hang.
func Send(req Request) (Response, error) {
	conn, err := net.DialTimeout("unix", SocketPath(), requestTimeout)
	if err != nil {
		return Response{}, fmt.Errorf("dock not reachable at %s: %w", SocketPath(), err)
	}
	defer conn.Close()

	conn.SetDeadline(deadline())

	data, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}

	if _, err := conn.Write(append(data, '\n')); err != nil {
		return Response{}, err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return Response{}, fmt.Errorf("no response: %w", err)
	}

	var resp Response
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return Response{}, fmt.Errorf("bad response: %w", err)
	}

	return resp, nil
}

// Ping reports whether a dock instance is accepting control commands.
func Ping() bool {
	resp, err := Send(Request{Cmd: CmdPing})
	return err == nil && resp.Ok
}
