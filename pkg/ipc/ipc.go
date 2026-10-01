package ipc

import (
	"log"
	"net"
	"strings"
	"time"
)

// Hyprland with a Lua config intercepts the raw socket command
// "dispatch <dispatcher> <args>" and evaluates "<dispatcher> <args>" as a Lua
// expression, which breaks on the space (')' expected near ...). The working
// protocol form is to pass the Lua call itself. Rewrite the two address-style
// dispatches hypr-dock uses to their Lua equivalents.
func luaDispatchCompat(cmd string) string {
	if strings.HasPrefix(cmd, "dispatch focuswindow address:") {
		addr := strings.TrimPrefix(cmd, "dispatch focuswindow address:")
		return `dispatch hl.dsp.focus({ window = "address:` + addr + `" })`
	}
	if strings.HasPrefix(cmd, "dispatch closewindow address:") {
		addr := strings.TrimPrefix(cmd, "dispatch closewindow address:")
		return `dispatch hl.dsp.window.close({ window = "address:` + addr + `" })`
	}
	return cmd
}

func Hyprctl(cmd string) (response []byte, err error) {
	conn, err := net.Dial("unix", getUnixSockAdress())
	if err != nil {
		return nil, err
	}

	message := []byte(luaDispatchCompat(cmd))
	_, err = conn.Write(message)
	if err != nil {
		return nil, err
	}

	response = make([]byte, 102400)
	n, err := conn.Read(response)
	if err != nil {
		return nil, err
	}

	defer conn.Close()

	return response[:n], nil
}

func InitHyprEvents() {
	for {
		unixConnect, err := net.DialUnix("unix", nil, getUnixSock2Adress())
		if err != nil {
			log.Printf("Failed to connect to Unix socket: %v. Retrying in 5 seconds...", err)
			time.Sleep(5 * time.Second)
			continue
		}
		defer unixConnect.Close()

		for {
			buffer := make([]byte, 10240)
			unixNumber, err := unixConnect.Read(buffer)
			if err != nil {
				log.Printf("Error reading from Unix socket: %v. Reconnecting...", err)
				break
			}

			events := splitEvent(string(buffer[:unixNumber]))
			for _, event := range events {
				go DispatchEvent(event)
			}
		}
	}
}
