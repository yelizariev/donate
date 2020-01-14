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
	"regexp"
	"strconv"
	"strings"
	"time"

	"code.dumpstack.io/lib/cryptocurrency"
	"github.com/google/go-github/v29/github"
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

func lookupPR(gh *github.Client, ctx context.Context,
	owner, project, commit string) (body string, found bool, err error) {

	pullRequests, _, err := gh.PullRequests.ListPullRequestsWithCommit(
		ctx, owner, project, commit, nil)
	if err != nil {
		return
	}

	for _, pr := range pullRequests {
		if pr.MergedAt == nil {
			continue
		}

		found = true
		body = *pr.Body
		break
	}
	return
}

func parseBTC(body string) (btc string) {
	re := regexp.MustCompile("BTC{([a-zA-Z0-9]*)}")
	match := re.FindStringSubmatch(body)
	if len(match) >= 2 {
		btc = match[1]
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
	w http.ResponseWriter, r *http.Request, donationAddress string) {

	repo, issue, err := parse(r.URL)
	if err != nil {
		log.Println(err)
		return
	}

	issueNo, err := strconv.Atoi(issue) // just additional sanity check
	if err != nil {
		log.Println(err)
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

	seed, address, err := issueGet(db, repo, issue)
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "repo/issue not found in database\n")
		return
	}

	// 1. Check that issue is closed
	ghIssue, _, err := gh.Issues.Get(ctx, owner, project, issueNo)
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "invalid repo/issue\n")
		return
	}
	if *ghIssue.State == "open" {
		fmt.Fprint(w, "issue is still open\n")
		return
	}

	// 2. Lookup for pull request that was close this issue
	events, _, err := gh.Issues.ListIssueEvents(ctx, owner, project, issueNo, nil)
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "something went wrong\n")
		return
	}

	btc := ""
	found := false
	for _, event := range events {
		if event.CommitID != nil {
			commit := *event.CommitID

			// 3. Check that there's bitcoin address in pull request
			var body string
			body, found, err = lookupPR(gh, ctx, owner, project, commit)
			if err != nil {
				return
			}

			if !found {
				continue
			}

			btc = parseBTC(body)
			log.Println("BTC:", btc)
			break
		}
	}

	if found {
		valid, err := cryptocurrency.Bitcoin.Validate(btc)
		if err != nil || !valid {
			fmt.Fprint(w, "invalid bitcoin address\n")
			return
		}
	}

	var tx string
	if !found || btc == address {
		// b. If no address then send to the donation address
		tx, err = cryptocurrency.Bitcoin.SendAll(seed, donationAddress)
		if err != nil {
			log.Println(err)
			fmt.Fprint(w, "something went wrong\n")
		}
		return
	}

	// a. If address exists just send all
	tx, err = cryptocurrency.Bitcoin.SendAll(seed, btc)
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "something went wrong\n")
		return
	}

	fmt.Fprintf(w, "%s\n", tx)
}

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
