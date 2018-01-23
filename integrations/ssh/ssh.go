package ssh

import (
	"fmt"
	"github.com/benchlabs/bub/core"
	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

type Tunnel struct {
	Local  *Endpoint
	Server *Endpoint
	Remote *Endpoint
	client *ssh.Client

	clientConfig *ssh.ClientConfig
	cfg          *core.Configuration
}

type Endpoint struct {
	Host string
	Port int
}

func Connect(cfg *core.Configuration, jumpHost, remoteHost string, localPort, remotePort int) (*Tunnel, error) {
	identityFile := ssh_config.Get(jumpHost, "IdentityFile")
	config := &ssh.ClientConfig{
		User: ssh_config.Get(jumpHost, "User"),
		Auth: []ssh.AuthMethod{
			agentAuth(),
			keyFileAuth(identityFile),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			checker := NewHostKeyChecker(NewHostKeyFile())
			return checker.Check(hostname, remote, key)
		},
	}
	localEndpoint := &Endpoint{
		Host: "localhost",
		Port: localPort,
	}
	jumpHostPort, err := strconv.Atoi(ssh_config.Get(jumpHost, "Port"))
	if err != nil {
		return nil, err
	}
	serverEndpoint := &Endpoint{
		Host: jumpHost,
		Port: jumpHostPort,
	}

	remoteEndpoint := &Endpoint{
		Host: remoteHost,
		Port: remotePort,
	}

	tunnel := &Tunnel{
		cfg:          cfg,
		clientConfig: config,
		Local:        localEndpoint,
		Server:       serverEndpoint,
		Remote:       remoteEndpoint,
	}

	err = tunnel.start()
	return tunnel, err
}

func agentAuth() ssh.AuthMethod {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func keyFileAuth(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

func (t *Tunnel) start() error {
	log.Printf("Connecting to %v through %v (local port: %v)...", t.Remote.Host, t.Server.Host, t.Local.Port)
	serverConn, err := ssh.Dial("tcp", t.Server.String(), t.clientConfig)
	t.client = serverConn
	if err != nil {
		log.Fatalf("Server dial error: %s\n", err)
	}

	listener, err := net.Listen("tcp", t.Local.String())
	if err != nil {
		return err
	}
	go func() {
		defer listener.Close()
		for {
			localConn, err := listener.Accept()
			if err != nil {
				log.Printf("Failed to accept traffic. %v", err)
			}
			remoteConn, err := serverConn.Dial("tcp", t.Remote.String())
			if err != nil {
				log.Fatalf("Remote dial error: %s\n", err)
			}
			go t.forward(localConn, remoteConn)
		}
	}()
	return nil
}

func (t *Tunnel) Close() error {
	return t.client.Close()
}

func (t *Tunnel) Command(cmd ...string) error {
	session, err := t.client.NewSession()
	if err != nil {
		return err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	go io.Copy(stdin, os.Stdin)
	return session.Run(strings.Join(cmd, " "))
}

func (t *Tunnel) CommandWithStrOutput(cmd ...string) (string, error) {
	stdout, err := t.CommandWithOutput(cmd...)
	content, err := ioutil.ReadAll(stdout)
	return strings.Trim(string(content), "\n"), err
}

func (t *Tunnel) CommandWithOutput(cmd ...string) (io.Reader, error) {
	session, err := t.client.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session: %s", err)
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return stdout, err
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return stdout, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		return stdout, err
	}
	go io.Copy(os.Stderr, stderr)
	go io.Copy(stdin, os.Stdin)
	return stdout, session.Run(strings.Join(cmd, " "))
}

func (t *Tunnel) forward(localConn net.Conn, remoteConn net.Conn) {
	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error: %s", err)
		}
	}
	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}
