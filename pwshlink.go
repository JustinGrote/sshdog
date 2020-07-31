package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/shiena/ansicolor"
	"golang.org/x/crypto/ssh"
)

func openPwshLink() {
	// Connect to the local SSH Server
	dbg.Debug("Connecting to localhost:2222")

	sc, err := net.Dial("tcp4", "localhost:2222")
	if err != nil {
		log.Fatalln(fmt.Printf("Failed connection to local ssh server: %s", err))
	}

	//Connect to the PWSH link
	cfg := &ssh.ClientConfig{
		User: "jgrote",
		Auth: []ssh.AuthMethod{
			ssh.Password("ncc1701EE"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", "pwsh.link:2222", cfg)
	if err != nil {
		log.Fatalln(fmt.Printf("Error connecting to pwsh.link: %s", err))
	}

	// We only care about the global requests to respond to keepalives, not the channel itself
	_, reqch, err := client.OpenChannel("keepalive", nil)
	go func() {
		//Reply to any global messages with empty failure reply, which is sufficient for server keepalives
		//https://tools.ietf.org/html/draft-ssh-global-requests-ok-00#section-4.1
		for {
			req := <-reqch
			if req.WantReply == true {
				err := req.Reply(false, nil)
				if err != nil {
					log.Fatalln(fmt.Printf("Failed to reply to server keepalive: %s", err))
				}
			}
		}
	}()

	sess, err := client.NewSession()
	if err != nil {
		log.Fatalln(fmt.Printf("Failed Requesting Session: %s", err))
	}
	sess.Stdout = ansicolor.NewAnsiColorWriter(os.Stdout)
	sess.Stderr = ansicolor.NewAnsiColorWriter(os.Stderr)
	in, _ := sess.StdinPipe()

	fmt.Fprint(in, "OK")
	err = sess.Shell()
	if err != nil {
		log.Fatalln(fmt.Printf("Failed Requesting Shell: %s", err))
	}
	// Start port Listener
	listenAddress := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 40022,
	}
	l, err := client.ListenTCP(listenAddress)

	if err != nil {
		log.Fatalln(fmt.Printf("Error requesting forwarded port from pwsh.link: %s", err))
	}
	c, err := l.Accept()

	if err != nil {
		log.Fatalln(fmt.Printf("Failed accepting connection from remote port listener: %s", err))
	}

	// Splice the streams together
	go copyAsync(c, sc, nil)
	go copyAsync(sc, c, nil)
}
