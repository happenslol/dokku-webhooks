package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"

	"github.com/boltdb/bolt"

	webhooks "github.com/happenslol/dokku-webhooks"
)

func listen() {
	usr, _ := user.Lookup("root")
	grp, _ := user.LookupGroup("root")

	if usr == nil || grp == nil {
		log.Fatal("user did not exist\n")
	}

	uid, _ := strconv.Atoi(usr.Uid)
	gid, _ := strconv.Atoi(grp.Gid)

	sock, err := net.Listen("unix", cmdSocket)
	if err != nil {
		log.Fatalf("could not create socket: %v\n", err)
	}

	log.Printf("listening on %s\n", cmdSocket)
	defer sock.Close()

	err = os.Chown(cmdSocket, uid, gid)
	if err != nil {
		log.Fatalf("could not set cmd socket owner: %v\n", err)
	}

	err = os.Chmod(cmdSocket, 0777)
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

	var res webhooks.Response

	switch cmd.T {
	case webhooks.CmdPing:
		res.Content = "up"
	case webhooks.CmdShowApp:

	case webhooks.CmdEnableApp:
		app := cmd.Args[0]
		// TODO(happens): Verify app
		// TODO(happens): Test for webhooks app lol

		_ = hookStorage.Update(func(tx *bolt.Tx) error {
			apps := tx.Bucket([]byte(enabledBucket))
			raw := apps.Get([]byte(app))
			enabled := raw != nil && string(raw) == ""

			if enabled {
				res.Content = "app was already enabled"
				return nil
			}

			err := apps.Put([]byte(app), []byte(""))
			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("failed to enable app: %v", err)
				return nil
			}

			res.Content = "app enabled"
			return nil
		})

	case webhooks.CmdDisableApp:
		app := cmd.Args[0]
		// TODO(happens): Verify app
		// TODO(happens): Test for webhooks app lol

		_ = hookStorage.Update(func(tx *bolt.Tx) error {
			apps := tx.Bucket([]byte(enabledBucket))
			raw := apps.Get([]byte(app))
			enabled := raw != nil && string(raw) == ""

			if !enabled {
				res.Content = "app was not enabled"
				return nil
			}

			err := apps.Delete([]byte(app))
			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("failed to disable app: %v", err)
				return nil
			}

			res.Content = "app disabled"
			return nil
		})

	case webhooks.CmdCreate:
		app, hook, command := cmd.Args[0], cmd.Args[1], cmd.Args[3]

		_ = hookStorage.Update(func(tx *bolt.Tx) error {
			appBucketStr := fmt.Sprintf("app/%s", app)
			appBucket, err := tx.CreateBucketIfNotExists([]byte(appBucketStr))

			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("could not create app bucket: %v", err)
				return nil
			}

			foundRaw := appBucket.Get([]byte(hook))
			if foundRaw != nil {
				res.Status = 1
				res.Content = "a hook with that name already exists"
				return nil
			}

			hookObj := hookData{
				Name:            hook,
				CommandTemplate: command,
				// TODO(happens): Parse command arguments in here for easier
				// validation during activation
			}

			ser, err := json.Marshal(hookObj)
			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("failed to serialize hook: %v", err)
				return nil
			}

			err = appBucket.Put([]byte(hook), ser)
			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("unable to save hook: %v", err)
				return nil
			}

			return nil
		})

	case webhooks.CmdDelete:
		app, hook := cmd.Args[0], cmd.Args[1]

		_ = hookStorage.Update(func(tx *bolt.Tx) error {
			appBucketStr := fmt.Sprintf("app/%s", app)
			appBucket := tx.Bucket([]byte(appBucketStr))
			if appBucket == nil {
				res.Status = 0
				res.Content = "hook does not exist"
				return nil
			}

			foundRaw := appBucket.Get([]byte(hook))
			if foundRaw == nil {
				res.Status = 0
				res.Content = "hook does not exist"
				return nil
			}

			err := appBucket.Delete([]byte(hook))
			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("failed to delete hook: %v", err)
				return nil
			}

			res.Content = "hook deleted"
			return nil
		})

	case webhooks.CmdSetSecret:
	case webhooks.CmdGenSecret:
	case webhooks.CmdShowSecret:

	case webhooks.CmdTrigger:
	case webhooks.CmdLogs:
	case webhooks.CmdQuit:
		done <- true
	}

	sendEncoded(c, res)
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
