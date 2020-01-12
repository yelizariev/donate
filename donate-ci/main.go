package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func getBalance(btc string) (balance float64, err error) {
	urlf := "https://api.blockcypher.com/v1/btc/main/addrs/%s/balance"
	resp, err := http.Get(fmt.Sprintf(urlf, btc))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct{ Balance float64 }
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return
	}

	balance = result.Balance / 100000000
	return
}

func genBody(btc string) (body string) {
	body = fmt.Sprintf("### Address for donations: "+
		"[%s](https://blockchair.com/bitcoin/address/%s).\n", btc, btc)

	balance, err := getBalance(btc)
	if err == nil {
		body += fmt.Sprintf("#### Current balance: %.8f BTC", balance)
	}
	body += "\n"

	body += "Usage:\n"
	body += "1. Specify this issue in commit message ([keywords]" +
		"(https://help.github.com/en/github/managing-your-work-on-" +
		"github/closing-issues-using-keywords)).\n"
	body += "2. Put to the body of pull request your BTC address in " +
		"the format: BTC{your_btc_address}.\n"
	body += "###### The default fee is 0% (someone who will solve this " +
		"issue will get all money without commission). " +
		"Consider donating to the [donation project]" +
		"(https://github.com/jollheef/donate) " +
		"itself, it'll help keep it work with zero fees.\n"
	return
}

func getAddr(gh *github.Client, ctx context.Context,
	owner, project, endpoint string, issueNo int) (btc string, err error) {

	url := fmt.Sprintf("%s/query?repo=github.com/%s/%s&issue=%d",
		endpoint, owner, project, issueNo)

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	btc = string(bytes) // note that in next versions here will be JSON
	btc = strings.TrimSpace(btc)
	return
}

func updateIssue(gh *github.Client, ctx context.Context,
	owner, project, endpoint string, issue *github.Issue) (err error) {

	number := *issue.Number
	btc, err := getAddr(gh, ctx, owner, project, endpoint, number)
	if err != nil {
		return
	}

	body := genBody(btc)

	comments, _, err := gh.Issues.ListComments(ctx, owner, project, number, nil)

	found := false
	for _, comment := range comments {
		if strings.Contains(*comment.Body, btc) {
			found = true
			newcomment := github.IssueComment{Body: &body}
			_, _, err = gh.Issues.EditComment(ctx, owner, project,
				*comment.ID, &newcomment)
			if err != nil {
				return
			}
		}
	}

	if !found {
		comment := github.IssueComment{Body: &body}
		_, _, err = gh.Issues.CreateComment(ctx, owner, project, number, &comment)
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

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	tx := string(bytes)
	tx = strings.TrimSpace(tx)

	if len(tx) != 64 { // TXID is always 32 bytes (64 characters)
		return
	}

	body := fmt.Sprintf("Tx: [%s](https://blockchair.com/bitcoin/transaction/%s)",
		tx, tx)

	number := *issue.Number
	comment := github.IssueComment{Body: &body}
	_, _, err = gh.Issues.CreateComment(ctx, owner, project, number, &comment)
	if err != nil {
		return
	}

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

func main() {
	log.SetFlags(log.Lshortfile)
	rand.Seed(time.Now().UnixNano())

	app := kingpin.New("donate-ci", "cryptocurrency donation CI cli")
	app.Author("Mikhail Klementev <root@dumpstack.io>")
	app.Version("0.0.0")

	token := app.Flag("token", "GitHub access token").Envar("GITHUB_TOKEN").Required().String()
	repo := app.Flag("repo", "GitHub repository").Envar("GITHUB_REPOSITORY").Required().String()
	endpoint := app.Flag("endpoint", "URL of donation server").Envar("DONATE_ENDPOINT").Default("https://donate.dumpstack.io").String()

	kingpin.MustParse(app.Parse(os.Args[1:]))

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
