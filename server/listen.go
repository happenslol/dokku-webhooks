package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"

	webhooks "github.com/happenslol/dokku-webhooks"
)

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
	defer sock.Close()

	err = os.Chown(cmdSocket, uid, gid)
	if err != nil {
		log.Fatalf("could not set cmd socket permissions: %v\n", err)
	}

	cons := make(chan net.Conn, 10)
	done := make(chan bool, 1)
	sigc := make(chan os.Signal, 1)

	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT)
	go func(c chan os.Signal) {
		sig := <-c
		log.Printf("received signal: %s\n", sig)
		done <- true
	}(sigc)

	go acceptIncoming(sock, cons)

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

func handleClient(c net.Conn, done chan<- bool) {
	// NOTE(happens): We always want to close this since
	// we only ever get one cmd and send one response
	defer c.Close()
	de := json.NewDecoder(c)

	var cmd webhooks.Cmd
	if err := de.Decode(&cmd); err != nil {
		log.Printf("unable to decode message: %v\n", err)
		return
	}

	log.Printf("received command: %v\n", cmd)

	switch cmd.T {
	case webhooks.CmdPing:
		var res webhooks.Response
		sendEncoded(c, res)
	case webhooks.CmdShowApp:
	case webhooks.CmdEnableApp:
	case webhooks.CmdDisableApp:
	case webhooks.CmdCreate:
	case webhooks.CmdDelete:
	case webhooks.CmdTrigger:
	case webhooks.CmdLogs:
	case webhooks.CmdQuit:
		done <- true
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

func sendEncoded(c net.Conn, msg webhooks.Response) {
	encoded, _ := json.Marshal(msg)
	c.Write(encoded)
}
