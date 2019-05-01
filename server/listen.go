package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"os/user"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/boltdb/bolt"
	"github.com/ryanuber/columnize"
	"golang.org/x/crypto/bcrypt"

	webhooks "github.com/happenslol/dokku-webhooks"
)

var argsRegex = regexp.MustCompile("\\$[a-zA-Z0-9-_.]+")

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

		case <-done:
			log.Printf("cmd socket listener shutting down\n")
			wg.Done()
			return
		}
	}
}

func handleClient(c net.Conn, done chan<- bool) {
	defer c.Close()
	de := json.NewDecoder(c)

	var cmd webhooks.Cmd
	if err := de.Decode(&cmd); err != nil {
		log.Printf("unable to decode message: %v\n", err)
		return
	}

	var res webhooks.Response

	switch cmd.T {
	case webhooks.CmdPing:
		res.Ok("up")
		sendEncoded(c, res)
		return

	case webhooks.CmdShowApp:
		log.Printf("running CmdShowApp with args %v\n", cmd.Args)
		app := cmd.Args[0]

		_ = hookStorage.View(func(tx *bolt.Tx) error {
			appBucketStr := fmt.Sprintf("app/%s", app)
			appBucket := tx.Bucket([]byte(appBucketStr))
			if appBucket == nil {
				res.Status = 0
				res.Content = "no webhooks for this app"
				return nil
			}

			hooks := []string{"NAME | COMMAND | LAST ACTIVATION"}
			_ = appBucket.ForEach(func(k []byte, v []byte) error {
				var hook hookData
				if err := json.Unmarshal(v, &hook); err != nil {
					// skip if we can't read it. should probably report something
					// or just delete it outright?
					return nil
				}

				timeStr := "never"
				if hook.LastActivation != nil {
					actTime := time.Unix(*hook.LastActivation, 0)
					timeStr = actTime.Format("2006-01-02 15:04:05")
				}

				hooks = append(hooks, fmt.Sprintf(
					"%s | %s | %s",
					hook.Name,
					hook.CommandTemplate,
					timeStr,
				))
				return nil
			})

			res.Content = columnize.SimpleFormat(hooks)
			return nil
		})

		sendEncoded(c, res)
		return

	case webhooks.CmdEnableApp:
		log.Printf("running CmdEnableApp with args %v\n", cmd.Args)
		app := cmd.Args[0]

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

		sendEncoded(c, res)
		return

	case webhooks.CmdDisableApp:
		log.Printf("running CmdDisableApp with args %v\n", cmd.Args)
		app := cmd.Args[0]

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

		sendEncoded(c, res)
		return

	case webhooks.CmdCreate:
		log.Printf("running CmdCreate with args %v\n", cmd.Args)
		app, hook, command := cmd.Args[0], cmd.Args[1], cmd.Args[2]

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

			hookArgs := argsRegex.FindAllString(command, -1)
			hookObj := hookData{
				Name:            hook,
				CommandTemplate: command,
				Args:            hookArgs,
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

			res.Content = fmt.Sprintf(
				"webhook created. endpoint: /%s/%s",
				app, hook,
			)
			return nil
		})

		sendEncoded(c, res)
		return

	case webhooks.CmdDelete:
		log.Printf("running CmdDelete with args %v\n", cmd.Args)
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

			res.Content = fmt.Sprintf(
				"webhook %s/%s deleted",
				app, hook,
			)
			return nil
		})

		sendEncoded(c, res)
		return

	case webhooks.CmdSetSecret:
		log.Printf("running CmdSetSecret with args %v\n", cmd.Args)
		app, secret, forceStr := cmd.Args[0], cmd.Args[1], cmd.Args[2]
		force := forceStr == "true"

		_ = hookStorage.Update(func(tx *bolt.Tx) error {
			secrets := tx.Bucket([]byte(secretsBucket))

			if !force && secrets.Get([]byte(app)) != nil {
				res.Status = 1
				res.Content = "secret is already set, please use `--force` if you want to overwrite it"
				return nil
			}

			encrypted, err := bcrypt.GenerateFromPassword([]byte(secret), 10)
			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("failed to encrypt secret: %v", err)
				return nil
			}

			err = secrets.Put([]byte(app), []byte(encrypted))
			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("failed to save secret: %v", err)
				return nil
			}

			res.Content = fmt.Sprintf(
				"set secret for %s: %s\n%s",
				app, secret,
				"you should save this somewhere, the plaintext can not be retrieved after this!",
			)

			return nil
		})

		sendEncoded(c, res)
		return

	case webhooks.CmdGenSecret:
		log.Printf("running CmdGenSecret with args %v\n", cmd.Args)
		app, forceStr, lengthStr := cmd.Args[0], cmd.Args[1], cmd.Args[2]
		force := forceStr == "true"
		length, err := strconv.Atoi(lengthStr)
		if err != nil {
			res.Status = 1
			res.Content = fmt.Sprintf("requested secret length is not a number: %s", lengthStr)
			sendEncoded(c, res)
			return
		}

		_ = hookStorage.Update(func(tx *bolt.Tx) error {
			secrets := tx.Bucket([]byte(secretsBucket))

			if !force && secrets.Get([]byte(app)) != nil {
				res.Status = 1
				res.Content = "secret is already set, please use `--force` if you want to overwrite it"
				return nil
			}

			gen, err := genSecret(length)
			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("failed to generate secret: %v", err)
				return nil
			}

			encrypted, err := bcrypt.GenerateFromPassword([]byte(gen), 10)
			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("failed to encrypt secret: %v", err)
				return nil
			}

			err = secrets.Put([]byte(app), []byte(encrypted))
			if err != nil {
				res.Status = 1
				res.Content = fmt.Sprintf("failed to save secret: %v", err)
				return nil
			}

			res.Content = fmt.Sprintf(
				"generated secret for %s: %s\n%s",
				app, gen,
				"you should save this somewhere, the plaintext can not be retrieved after this!",
			)

			return nil
		})

		sendEncoded(c, res)
		return

	case webhooks.CmdTrigger:
		log.Printf("running CmdTrigger with args %v\n", cmd.Args)
		res.Content = "not implemented"
		sendEncoded(c, res)
		return
	case webhooks.CmdLogs:
		log.Printf("running CmdLogs with args %v\n", cmd.Args)
		res.Content = "not implemented"
		sendEncoded(c, res)
		return
	case webhooks.CmdQuit:
		log.Printf("running CmdQuit with args %v\n", cmd.Args)
		res.Status = 0
		res.Content = "shutting down"
		sendEncoded(c, res)
		done <- true
		return
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

func genSecret(length int) (string, error) {
	result := ""

	for {
		if len(result) >= length {
			return result, nil
		}
		num, err := rand.Int(rand.Reader, big.NewInt(int64(127)))
		if err != nil {
			return "", err
		}
		n := num.Int64()

		// Make sure that the number/byte/letter is inside
		// the range of printable ASCII characters (excluding space and DEL)
		if n > 32 && n < 127 {
			result += string(n)
		}
	}
}
