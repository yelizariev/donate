// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v29/github"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/umpc/go-sortedmap"
	"github.com/umpc/go-sortedmap/desc"
	"golang.org/x/oauth2"
)

func main() {
	db, err := leveldb.OpenFile(os.Getenv("DASHBOARD_DB_PATH"), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	gh := github.NewClient(tc)

	http.HandleFunc("/put", func(w http.ResponseWriter, r *http.Request) {
		err = putHandler(db, gh, ctx, w, r)
		if err != nil {
			log.Println(err)
		}
	})

	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		err = apiHandler(db, w, r)
		if err != nil {
			log.Println(err)
		}
	})

	http.HandleFunc("/redirect", redirectHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err = indexPage(db, w, r)
		if err != nil {
			log.Println(err)
		}
	})

	log.Fatal(http.ListenAndServe(":8040", nil))
}

type Sum struct {
	Cents    int
	Unixtime int64
}

func putHandler(db *leveldb.DB, gh *github.Client, ctx context.Context,
	w http.ResponseWriter, r *http.Request) (err error) {

	// curl '.../put?url=github.com/jollheef/donate/issues/9&sum=1337&key=ACCESS_KEY'
	keyValues, ok := r.URL.Query()["key"]
	if !ok || len(keyValues[0]) < 1 {
		return
	}
	strkey := string(keyValues[0])

	urlValues, ok := r.URL.Query()["url"]
	if !ok || len(urlValues[0]) < 1 {
		return
	}
	strurl := string(urlValues[0])

	// sum in cents
	sumValues, ok := r.URL.Query()["sum"]
	if !ok || len(sumValues[0]) < 1 {
		return
	}
	strsum := string(sumValues[0])
	sum, err := strconv.Atoi(strsum)
	if err != nil {
		return
	}

	err = check(gh, ctx, strurl, strkey, sum)
	if err != nil {
		// fuck off definition: to leave or go away, used
		// especially as a rude way of telling someone to
		// go away.
		return
	}

	raw, err := json.Marshal(Sum{Cents: sum, Unixtime: time.Now().Unix()})
	if err != nil {
		return
	}

	err = db.Put([]byte(strurl), raw, nil)
	return
}

func apiHandler(db *leveldb.DB, w http.ResponseWriter, r *http.Request) (err error) {
	var repo string
	values, ok := r.URL.Query()["url"]
	if !ok || len(values[0]) < 1 {
		repo = "all"
	} else {
		repo = string(values[0])
	}

	kv, err := getAll(db)
	if err != nil {
		return
	}

	iter, err := kv.IterCh()
	if err != nil {
		return
	}
	defer iter.Close()

	type issue struct {
		URL string
		USD string
	}

	var output struct {
		Issues []issue
	}

	for rec := range iter.Records() {
		if repo != "all" && !strings.HasPrefix(rec.Key.(string), repo) {
			continue
		}

		usd := float64(rec.Val.(int)) / float64(100)
		url := rec.Key.(string)

		output.Issues = append(output.Issues, issue{
			URL: url,
			USD: fmt.Sprintf("%.02f", usd),
		})
	}

	return json.NewEncoder(w).Encode(output)
}

func indexPage(db *leveldb.DB, w http.ResponseWriter, r *http.Request) (err error) {
	kv, err := getAll(db)
	if err != nil {
		return
	}

	iter, err := kv.IterCh()
	if err != nil {
		return
	}
	defer iter.Close()

	fmt.Fprint(w, `<html>`)
	fmt.Fprint(w, `<head>`)
	fmt.Fprint(w, `<meta charset="UTF-8">`)
	fmt.Fprint(w, `<title>list of bounties</title>`)
	fmt.Fprint(w, `</head>`)
	fmt.Fprint(w, `<link type="text/css" rel="stylesheet" `+
		`href="https://dumpstack.io/css/style.css">`)
	fmt.Fprint(w, `<body>`)
	fmt.Fprint(w, `<h1>list of bounties</h1>`)

	fmt.Fprint(w, "<ul>")
	for rec := range iter.Records() {
		ft := "<li>$%.02f — <a href=\"https://%s\">%s</a></li>\n"
		usd := float64(rec.Val.(int)) / float64(100)
		fmt.Fprintf(w, ft, usd, rec.Key.(string), rec.Key.(string))
	}
	fmt.Fprint(w, "</ul>")

	fmt.Fprint(w, `</body>`)
	fmt.Fprint(w, `</html>`)
	return
}

func getAll(db *leveldb.DB) (kv *sortedmap.SortedMap, err error) {
	kv = sortedmap.New(4, desc.Int)

	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		var sum Sum
		err = json.Unmarshal(value, &sum)
		if err != nil {
			log.Println("getAll json.Unmarshal", err)
			continue
		}

		// ignore zero values
		if sum.Cents == 0 {
			continue
		}

		// ignore values older than 24 hours (24 * 60 * 60 = 86400)
		if sum.Unixtime+86400 < time.Now().Unix() {
			err = db.Delete(key, nil)
			if err != nil {
				log.Println("getAll db.Delete", err)
				continue
			}
		}

		kv.Insert(string(key), sum.Cents)
	}

	err = iter.Error()
	return
}

// Note that even if here you can see a lot of checks, this was never
// supposed to be secure and without fake data.
//
// There's still a ways to put fake bounty, hovewer, we can be sure that
// there will be no XSS and links outside of repository' issues.
//
// It's just a way to get rid of completely stupid things. Anyway, abusers
// will be banned.
//
func check(gh *github.Client, ctx context.Context, url, key string, sum int) (err error) {
	owner, project, issueNo, err := parseURL(url)
	if err != nil {
		return
	}

	// Check for whitelist
	hashedAccessToken, exists := whitelist[owner]
	if !exists {
		err = errors.New(owner + " not in whitelist")
		return
	}

	// Check ACCESS_TOKEN
	if sha256sum(key) != hashedAccessToken {
		err = errors.New("invalid access token for " + owner)
		return
	}

	comments, _, err := gh.Issues.ListComments(ctx, owner, project, issueNo, nil)
	if err != nil {
		return
	}
	found := false
	for _, comment := range comments {
		if *comment.User.Login != "github-actions[bot]" {
			continue
		}

		fsum := float64(sum) / 100
		total := fmt.Sprintf("Total $%.02f", fsum)

		log.Println("look for", fsum, "in", owner, project, issueNo)

		if strings.Contains(*comment.Body, total) {
			found = true
			break
		}
	}
	if !found {
		err = errors.New("invalid url")
	}
	return
}

func parseURL(url string) (owner, project string, issueNo int, err error) {
	// only alphanumeric, underscore and dash
	re := regexp.MustCompile("^[\\./a-zA-Z0-9_-]*$")
	valid := re.MatchString(url)
	if !valid {
		err = errors.New("invalid url")
		return
	}

	// github.com/jollheef/donate/issues/9
	fields := strings.Split(url, "/")
	if len(fields) != 5 {
		err = errors.New("invalid url")
		return
	}

	// [github.com]/jollheef/donate/issues/9
	if fields[0] != "github.com" {
		// no support of anything else yet
		err = errors.New("invalid url")
		return
	}

	// github.com/[jollheef]/donate/issues/9
	if len(fields[1]) > 39 {
		// Github has a max username length of 39 characters
		err = errors.New("invalid url")
		return
	}
	owner = fields[1]

	// github.com/jollheef/[donate]/issues/9
	if len(fields[2]) > 100 {
		// Github has a max repository name length of 100 characters
		err = errors.New("invalid url")
		return
	}
	project = fields[2]

	// github.com/jollheef/donate/[issues]/9
	if fields[3] != "issues" {
		err = errors.New("invalid url")
		return
	}

	// github.com/jollheef/donate/issues/[9]
	issueNo, err = strconv.Atoi(fields[4])
	if err != nil {
		err = errors.New("invalid issue number")
		return
	}

	return
}

func sha256sum(data string) string {
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	values, ok := r.URL.Query()["url"]
	if !ok || len(values[0]) < 1 {
		return
	}
	url := string(values[0])

	_, _, _, err := parseURL(url)
	if err != nil {
		return
	}

	body := "<head>"
	body += "<meta http-equiv='refresh' content='0; URL=https://" + url + "'>"
	body += "</head>"

	fmt.Fprint(w, body)
}
