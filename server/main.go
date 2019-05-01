package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/boltdb/bolt"
	dokku "github.com/dokku/dokku/plugins/common"
)

var jobStorage *bolt.DB
var hookStorage *bolt.DB
var wg sync.WaitGroup

type hookData struct {
	Name            string
	CommandTemplate string
	Args            []string
	LastActivation  *int64
}

func (h hookData) GetCmd(args map[string]string) (string, error) {
	result := h.CommandTemplate
	missing := []string{}

	for _, arg := range h.Args {
		k := fmt.Sprintf("#%s", arg)
		val, ok := args[k]
		if !ok {
			missing = append(missing, arg)
			continue
		}

		result = strings.ReplaceAll(result, k, val)
	}

	if len(missing) > 0 {
		all := strings.Join(missing, ", ")
		e := fmt.Sprintf("missing arguments: %s", all)
		return "", errors.New(e)
	}

	return result, nil
}

const (
	secretsBucket = "secrets"
	enabledBucket = "enabled"
	storageDir    = "/app/storage"

	dokkuSocket = "/app/storage/dokku.sock"
	cmdSocket   = "/app/storage/cmd.sock"

	jobStoragePath  = "/app/storage/jobs.db"
	hookStoragePath = "/app/storage/hooks.db"
)

func main() {
	if !dokku.DirectoryExists(storageDir) {
		log.Fatalf("storage dir should exist: %s\n", storageDir)
	}

	if _, err := os.Stat(dokkuSocket); err != nil {
		log.Fatalf("dokku daemon socket should exist: %s\n", dokkuSocket)
	}

	if _, err := os.Stat(cmdSocket); err == nil {
		err = os.Remove(cmdSocket)
		if err != nil {
			log.Fatalf("could not remove old cmd socket: %s\n", err)
		}
	}

	var err error

	jobStorage, err = bolt.Open(jobStoragePath, 0777, nil)
	if err != nil {
		log.Fatalf("error opening job storage: %v\n", err)
	}
	defer jobStorage.Close()

	hookStorage, err = bolt.Open(hookStoragePath, 0777, nil)
	if err != nil {
		log.Fatalf("error opening hook storage: %v\n", err)
	}
	defer hookStorage.Close()

	_ = hookStorage.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte(secretsBucket))
		tx.CreateBucketIfNotExists([]byte(enabledBucket))
		return nil
	})

	wg.Add(2)

	go serve()
	go listen()

	wg.Wait()
}

func sendDokkuCmd(cmd string) {
	// TODO(happens): logging for this
	c, err := net.Dial("unix", dokkuSocket)
	if err != nil {
		return
	}
	defer c.Close()

	_, err = c.Write([]byte(cmd))
	if err != nil {
		return
	}
}
