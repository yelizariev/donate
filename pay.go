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
	"regexp"
	"strconv"
	"strings"

	c "code.dumpstack.io/lib/cryptocurrency"
	"github.com/google/go-github/v29/github"

	"code.dumpstack.io/tools/donate/database"
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

func findAddress(body, symbol string) (address string) {
	re := regexp.MustCompile(strings.ToUpper(symbol) + "{([a-zA-Z0-9]*)}")
	match := re.FindStringSubmatch(body)
	if len(match) >= 2 {
		address = match[1]
	}
	return
}

type userWallet struct {
	// Type is Bitcoin/Ethereum/etc.
	Type c.Cryptocurrency
	// Found address in pull request body or not
	Found bool
	// Tx represents transaction
	Tx string

	database.Wallet
}

func findWallets(body string) (wallets []userWallet) {
	for _, cc := range c.Cryptocurrencies {
		wallet := userWallet{Type: cc, Found: false}

		address := findAddress(body, cc.Symbol())
		if address != "" {
			wallet.Found = true
			wallet.Address = address
		}

		wallets = append(wallets, wallet)
	}
	return
}

func payHandler(db *sql.DB, gh *github.Client, ctx context.Context,
	w http.ResponseWriter, r *http.Request,
	defaultDests map[c.Cryptocurrency]string) (err error) {

	issue := database.NewIssue()
	var issueS string
	issue.Repo, issueS, err = parse(r.URL)
	if err != nil {
		log.Println(err)
		return
	}

	issue.ID, err = strconv.Atoi(issueS)
	if err != nil {
		log.Println(err)
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

	err = database.GetWallets(db, &issue, database.ShowSeed)
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "repo/issue not found in database\n")
		return
	}

	// 1. Check that issue is closed
	ghIssue, _, err := gh.Issues.Get(ctx, owner, project, issue.ID)
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
	events, _, err := gh.Issues.ListIssueEvents(ctx, owner, project, issue.ID, nil)
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "{}")
		return
	}

	var wallets []userWallet
	for _, event := range events {
		if event.CommitID != nil {
			commit := *event.CommitID

			// 3. Check that there's pull request
			var body string
			var found bool
			body, found, err = lookupPR(gh, ctx, owner, project, commit)
			if err != nil {
				return
			}
			if !found {
				continue
			}

			// Looking for all cryptocurrency wallets
			wallets = findWallets(body)
			break
		}
	}

	// No pull request was found, create dummy wallets
	if len(wallets) == 0 {
		for _, cc := range c.Cryptocurrencies {
			wallet := userWallet{Type: cc, Found: false}
			wallets = append(wallets, wallet)
		}
	}

	transactions := make(map[c.Cryptocurrency]string)
	for _, wallet := range wallets {
		if !wallet.Found {
			// b. If no address then send to the donation address
			address := defaultDests[wallet.Type]
			// Note that we're getting seed from the issue' wallet
			seed := issue.Wallets[wallet.Type].Seed
			tx, err := wallet.Type.SendAll(seed, address)
			if err != nil {
				log.Println("sendall error", err)
				err = nil
			}
			log.Print("tx -> default dest:", tx)
			// We don't show this transaction to user, to
			// avoid confusion. Of course, those transactions
			// are shown in the blockchain explorer, if someone
			// wants to know.
			continue
		}

		valid, err := wallet.Type.Validate(wallet.Address)
		if err != nil {
			// Error here does not mean that address is invalid
			// Do not send to anyone in this case
			err = nil
			log.Println("validate error", wallet.Address, err)
			continue
		}

		if valid {
			// Note that we're getting seed from the issue' wallet
			seed := issue.Wallets[wallet.Type].Seed
			tx, err := wallet.Type.SendAll(seed, wallet.Address)
			if err != nil {
				log.Println("sendall error", err)
				err = nil
			}
			transactions[wallet.Type] = tx
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(transactions)
	return
}
