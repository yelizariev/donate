// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"code.dumpstack.io/lib/cryptocurrency"
	"github.com/google/go-github/v29/github"

	"code.dumpstack.io/tools/donate/database"
)

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
		issues, err := database.IssueAll(db, repo)
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

	exists, err := database.IssueExists(db, repo, issue)
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

		err = database.IssueAdd(db, repo, issue, seed, address)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Fprintf(w, "%s\n", address)
		return
	}

	_, address, err := database.IssueGet(db, repo, issue)
	fmt.Fprintf(w, "%s\n", address)
	return
}
