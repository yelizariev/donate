// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"code.dumpstack.io/lib/cryptocurrency"
	"github.com/google/go-github/github"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func parse(url *url.URL) (repo, issue string, err error) {
	values, ok := url.Query()["repo"]
	if !ok || len(values[0]) < 1 {
		err = errors.New("No repo specified")
		return
	}
	repo = values[0]

	issue = "all"
	values, ok = url.Query()["issue"]
	if ok && len(values[0]) >= 1 {
		issue = values[0]
	}
	return
}

func queryHandler(db *sql.DB, gh *github.Client, ctx context.Context,
	w http.ResponseWriter, r *http.Request) {

	repo, issue, err := parse(r.URL)
	if err != nil {
		log.Println(err)
		return
	}

	if !strings.HasPrefix(repo, "github.com/") {
		fmt.Fprint(w, "non-github repos are not supported yet\n")
		return
	}

	fields := strings.Split(repo, "/")
	if len(fields) != 3 {
		fmt.Fprint(w, "invalid repo\n")
		return
	}
	// fields[0] is 'github.com'
	owner := fields[1]
	project := fields[2]

	if issue == "all" {
		issues, err := issueAll(db, repo)
		if err != nil {
			log.Println(err)
			return
		}

		js, err := json.Marshal(issues)
		if err != nil {
			log.Println(err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}

	issueNo, err := strconv.Atoi(issue)
	if err != nil {
		fmt.Fprint(w, "invalid issue\n")
		log.Println(err)
		return
	}

	exists, err := issueExists(db, repo, issue)
	if err != nil {
		log.Println(err)
		return
	}
	if !exists {
		// Check that issue is really exists on GitHub
		ghIssue, _, err := gh.Issues.Get(ctx, owner, project, issueNo)
		if err != nil {
			log.Println(err)
			fmt.Fprint(w, "invalid repo/issue\n")
			return
		}
		if *ghIssue.State != "open" {
			fmt.Fprint(w, "issue is not open\n")
			return
		}

		seed, address, err := cryptocurrency.Bitcoin.GenWallet()
		if err != nil {
			log.Println(err)
			return
		}

		err = issueAdd(db, repo, issue, seed, address)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Fprintf(w, "%s\n", address)
		return
	}

	_, address, err := issueGet(db, repo, issue)
	fmt.Fprintf(w, "%s\n", address)
	return
}

func payHandler(db *sql.DB, gh *github.Client, ctx context.Context,
	w http.ResponseWriter, r *http.Request) {

	repo, issue, err := parse(r.URL)
	if err != nil {
		log.Println(err)
		return
	}

	_, err = strconv.Atoi(issue) // just additional sanity check
	if err != nil {
		log.Println(err)
		return
	}

	seed, address, err := issueGet(db, repo, issue)
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "repo/issue not found")
		return
	}

	// TODO
	_ = seed
	_ = address

	// 1. Check that issue is closed

	// 2. Lookup for pull request that was close this issue

	// 3. Check that there's bitcoin address in pull request
	//    a. If address exists just send all
	//    b. If no address then send to random issue of the same project

	fmt.Fprint(w, "not implemented yet")
}

func main() {
	log.SetFlags(log.Lshortfile)
	rand.Seed(time.Now().UnixNano())

	app := kingpin.New("donate", "cryptocurrency donation daemon")
	app.Author("Mikhail Klementev <root@dumpstack.io>")
	app.Version("0.0.0")

	database := app.Flag("database", "Path to database").Required().String()
	token := app.Flag("token", "GitHub access token").Required().String()

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
		payHandler(db, client, ctx, w, r)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
