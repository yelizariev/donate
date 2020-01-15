// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"code.dumpstack.io/lib/cryptocurrency"
	"github.com/google/go-github/v29/github"
)

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
