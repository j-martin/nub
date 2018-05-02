package ssh

import (
	"fmt"
	"github.com/j-martin/bub/utils"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

type SSH interface {
	Connect() error
	Close() error
}

type Tunnel struct {
	LocalPort  int
	RemoteHost string
	RemotePort int
}

type Connection struct {
	JumpHost string
	Command  string
	Tunnels  map[string]Tunnel
	process  *os.Process
}

func (s *Connection) Connect() error {
	args := []string{
		"-o", "ExitOnForwardFailure yes",
		"-N",
	}
	for _, h := range s.Tunnels {
		args = append(args, "-L", fmt.Sprintf("%v:%v:%v", h.LocalPort, h.RemoteHost, h.RemotePort))
	}
	args = append(args, s.JumpHost)
	log.Printf("Connecting: ssh %v", strings.Join(args, " "))
	tunnel := exec.Command("ssh", args...)
	tunnel.Stderr = os.Stderr
	err := tunnel.Start()
	if err != nil {
		return err
	}
	log.Print("Waiting for tunnel(s)...")
	for _, h := range s.Tunnels {
		for {
			time.Sleep(20 * time.Millisecond)
			if IsListening(h.LocalPort) {
				break
			}
		}
	}
	s.process = tunnel.Process
	return nil
}

func (s *Connection) Close() error {
	return s.process.Kill()
}

func GetPort() int {
	for {
		port := utils.Random(40000, 60000)
		if !IsListening(port) {
			return port
		}
	}
}

func IsListening(port int) bool {
	_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%v", port))
	if err != nil {
		return false
	}
	return true
}
