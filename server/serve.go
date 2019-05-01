package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/boltdb/bolt"
	"github.com/go-chi/chi"
	"golang.org/x/crypto/bcrypt"
)

type ctxKey string

const (
	ctxApp  ctxKey = "app"
	ctxHook ctxKey = "hook"
)

func serve() {
	r := chi.NewRouter()
	r.Route("/{app}/{hook}", func(r chi.Router) {
		r.Use(validateApp)
		r.Use(validateSecret)
		r.Use(addHookContext)

		r.Post("/", executeHook)
	})

	r.Route("/health", func(r chi.Router) {
		r.Get("/", reportHealth)
	})

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = ":3000"
	} else {
		port = fmt.Sprintf(":%s", port)
	}

	fmt.Printf("listening on %s", port)

	go func() { http.ListenAndServe(port, r) }()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM, syscall.SIGABRT)

	<-sigc
	fmt.Printf("server shutting down\n")
	wg.Done()
}

func validateSecret(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		app := ctx.Value(ctxApp).(string)

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
			// anything more specific than 401 at this point, for
			// security reasons
			http.Error(w, http.StatusText(401), 401)
			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(found), []byte(pw))
		if len(found) == 0 || err != nil {
			http.Error(w, http.StatusText(401), 401)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func validateApp(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app := chi.URLParam(r, "app")

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

		ctx := context.WithValue(r.Context(), ctxApp, app)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func addHookContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		app := ctx.Value(ctxApp).(string)
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

		ctx = context.WithValue(ctx, ctxHook, found)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func executeHook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	hook, hookOk := ctx.Value(ctxHook).(*hookData)
	app, appOk := ctx.Value(ctxApp).(string)

	if !hookOk || !appOk {
		// NOTE(happens): This should not be able to happen since
		// the middleware will abort if there is no hook or app
		http.Error(w, http.StatusText(500), 500)
		return
	}

	query := r.URL.Query()
	params := make(map[string]string)

	params["app"] = app
	for k, _ := range query {
		params[k] = query.Get(k)
	}

	cmd, err := hook.GetCmd(params)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	fmt.Printf("executing command: %s\n", cmd)
	go sendDokkuCmd(cmd)

	w.WriteHeader(202)
	w.Write([]byte(http.StatusText(202)))
}

func reportHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("up"))
}
