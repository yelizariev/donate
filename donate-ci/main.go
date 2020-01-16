// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	c "code.dumpstack.io/lib/cryptocurrency"
	"code.dumpstack.io/tools/donate/database"
)

func getIssue(owner, project, endpoint string, issueNo int) (
	issue database.Issue, err error) {

	url := fmt.Sprintf("%s/query?repo=github.com/%s/%s&issue=%d",
		endpoint, owner, project, issueNo)

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	issue = database.NewIssue()
	err = json.NewDecoder(resp.Body).Decode(&issue)
	return
}

func updateIssue(gh *github.Client, ctx context.Context,
	owner, project, endpoint string, ghIssue *github.Issue) (err error) {

	number := *ghIssue.Number
	issue, err := getIssue(owner, project, endpoint, number)
	if err != nil {
		return
	}

	body := genBody(issue)

	comments, _, err := gh.Issues.ListComments(ctx, owner, project, number, nil)

	found := false
	for _, comment := range comments {
		if strings.Contains(*comment.Body, issue.Wallets[c.Bitcoin].Address) {
			found = true
			newcomment := github.IssueComment{Body: &body}
			if !dryRun {
				_, _, err = gh.Issues.EditComment(ctx,
					owner, project, *comment.ID, &newcomment)
			} else {
				log.Println("old body:")
				fmt.Println(*comment.Body)
				fmt.Println()
				log.Println("new body:")
				fmt.Println(body)
			}
			if err != nil {
				return
			}
		}
	}

	if !found {
		comment := github.IssueComment{Body: &body}
		if !dryRun {
			_, _, err = gh.Issues.CreateComment(ctx,
				owner, project, number, &comment)
		}
		if err != nil {
			return
		}
	}
	return
}

func triggerPayout(gh *github.Client, ctx context.Context,
	owner, project, endpoint string, issue *github.Issue) (err error) {

	url := fmt.Sprintf("%s/pay?repo=github.com/%s/%s&issue=%d",
		endpoint, owner, project, *issue.Number)

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	transactions := make(map[c.Cryptocurrency]string)
	err = json.NewDecoder(resp.Body).Decode(&transactions)
	if err != nil {
		return
	}

	if len(transactions) == 0 {
		return
	}

	var body string
	valid := false
	for cc, tx := range transactions {
		if tx == "" {
			continue
		}
		valid = true

		var api string
		switch cc {
		case c.Bitcoin:
			api = "https://blockchair.com/bitcoin/transaction"
		case c.Ethereum:
			api = "https://blockchair.com/ethereum/transaction"
		default:
			log.Println("not supported transaction", cc, tx)
			continue
		}
		symbol := strings.ToUpper(cc.Symbol())
		body += fmt.Sprintf("- %s: [%s](%s/%s)\n", symbol, tx, api, tx)
	}

	if !valid {
		log.Println("no valid transactions found")
		return
	}

	body = "Payout transactions:\n" + body

	number := *issue.Number
	comment := github.IssueComment{Body: &body}
	_, _, err = gh.Issues.CreateComment(ctx, owner, project, number, &comment)
	return
}

func walkIssue(gh *github.Client, ctx context.Context,
	owner, project, endpoint string, issue *github.Issue) (err error) {

	if issue.ClosedAt != nil {
		if issue.ClosedAt.Before(time.Now().Add(-24 * time.Hour)) {
			// ignore issues that have closed more than one day ago
			return
		}
	}

	if *issue.State == "open" {
		err = updateIssue(gh, ctx, owner, project, endpoint, issue)
	} else {
		err = triggerPayout(gh, ctx, owner, project, endpoint, issue)
	}
	return
}

func walk(gh *github.Client, ctx context.Context, repo, endpoint string) (err error) {
	// GITHUB_REPOSITORY=jollheef/test-repo-please-ignore
	fields := strings.Split(repo, "/")
	if len(fields) != 2 {
		err = errors.New("invalid repo")
		return
	}
	owner := fields[0]
	project := fields[1]

	options := github.IssueListByRepoOptions{State: "all"}
	issues, _, err := gh.Issues.ListByRepo(ctx, owner, project, &options)
	for _, issue := range issues {
		err = walkIssue(gh, ctx, owner, project, endpoint, issue)
		if err != nil {
			log.Println(err)
			err = nil // do not exit
		}
	}
	return
}

var dryRun = false

func main() {
	log.SetFlags(log.Lshortfile)
	rand.Seed(time.Now().UnixNano())

	app := kingpin.New("donate-ci", "cryptocurrency donation CI cli")
	app.Author("Mikhail Klementev <root@dumpstack.io>")
	app.Version("3.1.0")

	token := app.Flag("token", "GitHub access token").Envar("GITHUB_TOKEN").Required().String()
	repo := app.Flag("repo", "GitHub repository").Envar("GITHUB_REPOSITORY").Required().String()
	endpoint := app.Flag("endpoint", "URL of donation server").Envar("DONATE_ENDPOINT").Default("https://donate.dumpstack.io").String()
	dry := app.Flag("dry-run", "Do not post any comments").Default("false").Bool()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	dryRun = *dry

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(ctx, ts)

	gh := github.NewClient(tc)

	err := walk(gh, ctx, *repo, *endpoint)
	if err != nil {
		log.Fatal(err)
	}
}
