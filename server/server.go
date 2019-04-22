package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"github.com/boltdb/bolt"
	"github.com/go-chi/chi"
)

var jobStorage *bolt.DB
var hookStorage *bolt.DB

func init() {
	var err error
	jobStorage, err = bolt.Open("jobs.db", 0600, nil)
	if err != nil {
		log.Fatalf("error opening job storage: %v\n", err)
	}

	hookStorage, err = bolt.Open("hooks.db", 0600, nil)
	if err != nil {
		log.Fatalf("error opening hook storage: %v\n", err)
	}
}

func main() {
	defer jobStorage.Close()
	defer hookStorage.Close()

	var wg sync.WaitGroup
	go serve(&wg)
	go listen(&wg)

	wg.Wait()
}

func serve(wg *sync.WaitGroup) {
	wg.Add(1)

	r := chi.NewRouter()
	r.Route("{appID}/{hookID}", func(r chi.Router) {
		r.Use(validateSecret)
		r.Use(hookContext)

		r.Post("/", executeHook)
	})

	http.ListenAndServe(":3000", r)
	wg.Done()
}

func listen(wg *sync.WaitGroup) {
	wg.Add(1)
	// TODO(happens): Connect to socket and listen to cli
	wg.Done()
}

func validateSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appID := chi.URLParam(r, "appID")
		b, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			http.Error(w, http.StatusText(500), 500)
			return
		}

		// TODO(happens): Does it make any difference if we make this
		// a string here, since it will be used as bytes by bcrypt anyways?
		pw := string(b)
		var found string

		err = hookStorage.View(func(tx *bolt.Tx) error {
			secrets := tx.Bucket([]byte("secrets"))
			if secrets == nil {
				return errors.New("secrets bucket not found")
			}

			foundRaw := secrets.Get([]byte(appID))
			if foundRaw == nil {
				return errors.New("app secret not found")
			}

			found = string(foundRaw)
			return nil
		})

		if err != nil {
			// NOTE(happens): We generally never want to return
			// anything more specific than 403 at this point, for
			// security reasons
			http.Error(w, http.StatusText(403), 403)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(found), []byte(pw))
		if len(found) == 0 || err != nil {
			http.Error(w, http.StatusText(403), 403)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type ctxKey string

type hook struct {
	Name            string
	CommandTemplate string
	Args            []string
}

func hookContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appID := chi.URLParam(r, "appID")
		hookID := chi.URLParam(r, "hookID")

		var found hook
		err := hookStorage.View(func(tx *bolt.Tx) error {
			appBucketStr := fmt.Sprintf("app/%s", appID)
			appBucket := tx.Bucket([]byte(appBucketStr))
			if appBucket == nil {
				e := fmt.Sprintf("app %s does not have any hooks", appID)
				return errors.New(e)
			}

			foundRaw := appBucket.Get([]byte(hookID))
			if foundRaw == nil {
				e := fmt.Sprintf("app %s has no hook named %s", appID, hookID)
				// TODO(happens): Print available hooks for app?
				return errors.New(e)
			}

			err := json.Unmarshal(foundRaw, &found)
			if err != nil {
				e := fmt.Sprintf("error reading hook data: %v, data:%v", err, foundRaw)
				return errors.New(e)
			}

			return nil
		})

		if err != nil {
			// TODO(happens): Correct error code and better description
			http.Error(w, http.StatusText(404), 404)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKey("hook"), found)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func executeHook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hook, ok := ctx.Value(ctxKey("hook")).(*hook)
	if !ok {
		// NOTE(happens): This should not be able to happen since
		// the middleware will abort if there is no hook
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// TODO(happens): Do the thing
	fmt.Printf("executing hook command: %v\n", hook)

	w.WriteHeader(202)
	w.Write([]byte(http.StatusText(202)))
}
