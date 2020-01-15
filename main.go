// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/google/go-github/v29/github"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	log.SetFlags(log.Lshortfile)
	rand.Seed(time.Now().UnixNano())

	app := kingpin.New("donate", "cryptocurrency donation daemon")
	app.Author("Mikhail Klementev <root@dumpstack.io>")
	app.Version("2.0.0")

	database := app.Flag("database", "Path to database").Envar("DONATE_DB_PATH").Required().String()
	token := app.Flag("token", "GitHub access token").Envar("GITHUB_TOKEN").Required().String()
	donationAddress := app.Flag("donation-address",
		"Set the address to which any not acquired donation will be sent").Envar(
		"DONATION_ADDRESS").Default(
		// default donation address is donating to this project
		"bc1q23fyuq7kmngrgqgp6yq9hk8a5q460f39m8nv87").String()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	db, err := openDatabase(*database)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	http.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		queryHandler(db, client, ctx, w, r)
	})

	http.HandleFunc("/pay", func(w http.ResponseWriter, r *http.Request) {
		payHandler(db, client, ctx, w, r, *donationAddress)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
