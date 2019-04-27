package main

import (
	"log"
	"os"
	"sync"

	"github.com/boltdb/bolt"
	dokku "github.com/dokku/dokku/plugins/common"
)

var jobStorage *bolt.DB
var hookStorage *bolt.DB
var wg sync.WaitGroup

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

	if !dokku.FileExists(dokkuSocket) {
		log.Fatalf("dokku daemon socket should exist: %s\n", dokkuSocket)
	}

	if _, err := os.Stat(cmdSocket); err == nil {
		err = os.Remove(cmdSocket)
		if err != nil {
			log.Fatalf("could not remove old cmd socket: %s\n", err)
		}
	}

	var err error

	jobStorage, err = bolt.Open("jobs.db", 0777, nil)
	if err != nil {
		log.Fatalf("error opening job storage: %v\n", err)
	}
	defer jobStorage.Close()

	hookStorage, err = bolt.Open("hooks.db", 0777, nil)
	if err != nil {
		log.Fatalf("error opening hook storage: %v\n", err)
	}
	defer hookStorage.Close()

	_ = hookStorage.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte(secretsBucket))
		tx.CreateBucketIfNotExists([]byte(enabledBucket))
		return nil
	})

	go serve()
	go listen()

	wg.Wait()
}
