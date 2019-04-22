package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"os/user"
	"strconv"
)

const (
	cmdQuit = "quit"
)

// Command represents an input received from the cli
// TODO(happens): Move this to a shared library
type Command struct {
	T    string
	Args []string
}

func listen() {
	wg.Add(1)

	usr, _ := user.Lookup("dokku")
	grp, _ := user.LookupGroup("dokku")
	uid, _ := strconv.Atoi(usr.Uid)
	gid, _ := strconv.Atoi(grp.Gid)

	sock, err := net.Listen("unix", cmdSocket)
	if err != nil {
		log.Fatalf("could not create socket: %v\n", err)
	}

	err = os.Chown(cmdSocket, uid, gid)
	if err != nil {
		log.Fatalf("could not set cmd socket permissions: %s\n", err)
	}

	cons := make(chan net.Conn, 10)
	go acceptIncoming(sock, cons)

	done := make(chan bool, 1)

	for {
		select {
		case con := <-cons:
			go handleClient(con, done)

		case _ = <-done:
			log.Printf("cmd socket listener shutting down\n")
			wg.Done()
			return
		}
	}
}

func handleClient(client net.Conn, done chan<- bool) {
	de := json.NewDecoder(client)
	for {
		var cmd Command
		de.Decode(&de)

		log.Printf("received command: %v\n", cmd)

		if cmd.T == cmdQuit {
			done <- true
			break
		}
	}
}

func acceptIncoming(sock net.Listener, cons chan<- net.Conn) {
	for {
		con, err := sock.Accept()
		if err != nil {
			continue
		}

		cons <- con
	}
}
