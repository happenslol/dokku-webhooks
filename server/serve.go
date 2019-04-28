package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/boltdb/bolt"
	dokku "github.com/dokku/dokku/plugins/common"
	"github.com/go-chi/chi"
	"golang.org/x/crypto/bcrypt"
)

func serve() {
	wg.Add(1)

	r := chi.NewRouter()
	r.Route("{app}/{hook}", func(r chi.Router) {
		r.Use(validateApp)
		r.Use(validateSecret)
		r.Use(addHookContext)

		r.Post("/", executeHook)
	})

	r.Route("health", func(r chi.Router) {
		r.Get("/", reportHealth)
	})

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = ":3000"
	} else {
		port = fmt.Sprintf(":%s", port)
	}

	log.Printf("listening on %s", port)
	http.ListenAndServe(port, r)
	wg.Done()
}

func validateSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		app := ctx.Value("app").(string)

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
			secrets := tx.Bucket([]byte(secretsBucket))
			if secrets == nil {
				return errors.New("secrets bucket not found")
			}

			foundRaw := secrets.Get([]byte(app))
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

type ctxKey struct {
	string
}

type hookData struct {
	Name            string
	CommandTemplate string
	Args            []string
}

func validateApp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app := chi.URLParam(r, "app")

		if err := dokku.VerifyAppName(app); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		enabled := false
		_ = hookStorage.View(func(tx *bolt.Tx) error {
			apps := tx.Bucket([]byte(enabledBucket))
			raw := apps.Get([]byte(app))
			enabled = raw != nil && string(raw) == ""

			return nil
		})

		if !enabled {
			// TODO(happens): Explain how to enable hooks?
			http.Error(w, "hooks are not enabled for this app", 400)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKey{"app"}, app)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func addHookContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		app := ctx.Value(ctxKey{"app"}).(string)
		hook := chi.URLParam(r, "hook")

		var found hookData
		err := hookStorage.View(func(tx *bolt.Tx) error {
			appBucketStr := fmt.Sprintf("app/%s", app)
			appBucket := tx.Bucket([]byte(appBucketStr))
			if appBucket == nil {
				e := fmt.Sprintf("app %s does not have any hooks", app)
				return errors.New(e)
			}

			foundRaw := appBucket.Get([]byte(hook))
			if foundRaw == nil {
				e := fmt.Sprintf("app %s has no hook named %s", app, hook)
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

		ctx = context.WithValue(ctx, ctxKey{"hook"}, found)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func executeHook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hook, ok := ctx.Value(ctxKey{"hook"}).(*hookData)
	if !ok {
		// NOTE(happens): This should not be able to happen since
		// the middleware will abort if there is no hook
		http.Error(w, http.StatusText(500), 500)
		return
	}

	// TODO(happens): Do the thing
	log.Printf("executing hook command: %v\n", hook)

	w.WriteHeader(202)
	w.Write([]byte(http.StatusText(202)))
}

func reportHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("up"))
}
