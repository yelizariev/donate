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

	c "code.dumpstack.io/lib/cryptocurrency"
	"github.com/google/go-github/v29/github"

	"code.dumpstack.io/tools/donate/database"
)

func queryHandler(db *sql.DB, gh *github.Client, ctx context.Context,
	w http.ResponseWriter, r *http.Request) {

	var err error

	issue := database.NewIssue()
	var issueS string
	issue.Repo, issueS, err = parse(r.URL)
	if err != nil {
		log.Println(err)
		return
	}

	if !strings.HasPrefix(issue.Repo, "github.com/") {
		fmt.Fprint(w, "non-github repos are not supported yet\n")
		return
	}

	fields := strings.Split(issue.Repo, "/")
	if len(fields) != 3 {
		fmt.Fprint(w, "invalid repo\n")
		return
	}
	// fields[0] is 'github.com'
	owner := fields[1]
	project := fields[2]

	if issueS == "all" {
		var issues []database.Issue
		issues, err = database.AllIssues(db, issue.Repo, database.HideSeed)
		if err != nil {
			log.Println(err)
			return
		}

		var js []byte
		js, err = json.Marshal(issues)
		if err != nil {
			log.Println(err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
		return
	}

	issue.ID, err = strconv.Atoi(issueS)
	if err != nil {
		fmt.Fprint(w, "invalid issue\n")
		log.Println(err)
		return
	}

	exists, err := database.IsExists(db, issue)
	if err != nil {
		log.Println(err)
		return
	}
	if !exists {
		// Check that issue is really exists on GitHub
		ghIssue, _, err := gh.Issues.Get(ctx, owner, project, issue.ID)
		if err != nil {
			log.Println(err)
			fmt.Fprint(w, "invalid repo/issue\n")
			return
		}
		if *ghIssue.State != "open" {
			fmt.Fprint(w, "issue is not open\n")
			return
		}

		err = genWallets(db, issue)
		if err != nil {
			log.Println(err)
			return
		}
	}

	err = database.GetWallets(db, &issue, database.HideSeed)
	if err != nil {
		log.Println(err)
		return
	}
	js, err := json.Marshal(issue)
	if err != nil {
		log.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func genWallets(db *sql.DB, issue database.Issue) (err error) {
	for _, cc := range c.Cryptocurrencies {
		var seed, address string
		seed, address, err = cc.GenWallet()
		if err != nil {
			return
		}

		issue.Wallets[cc] = database.Wallet{
			Seed:    seed,
			Address: address,
		}
	}

	return database.Add(db, issue)
}
