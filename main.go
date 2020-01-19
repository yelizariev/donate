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

	c "code.dumpstack.io/lib/cryptocurrency"
	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"code.dumpstack.io/tools/donate/database"
)

func main() {
	log.SetFlags(log.Lshortfile)
	rand.Seed(time.Now().UnixNano())

	app := kingpin.New("donate", "cryptocurrency donation daemon")
	app.Author("Mikhail Klementev <root@dumpstack.io>")
	app.Version("3.2.1")

	databasePath := app.Flag("database", "Path to database").Envar("DONATE_DB_PATH").Required().String()
	token := app.Flag("token", "GitHub access token").Envar("GITHUB_TOKEN").Required().String()
	donationAddressBTC := app.Flag("donation-address-btc",
		"Set the Bitcoin address to which any not acquired donation will be sent").Envar(
		"DONATION_ADDRESS_BTC").Default(
		// default donation address is donating to this project
		"bc1q23fyuq7kmngrgqgp6yq9hk8a5q460f39m8nv87").String()
	donationAddressETH := app.Flag("donation-address-eth",
		"Set the Ethereum address to which any not acquired donation will be sent").Envar(
		"DONATION_ADDRESS_ETH").Default(
		// default donation address is donating to this project
		"0xD2237129937E40b32db36Cda0Ae2c82B5ceD2380").String()
	donationAddressADA := app.Flag("donation-address-ada",
		"Set the Cardano address to which any not acquired donation will be sent").Envar(
		"DONATION_ADDRESS_ADA").Default(
		// default donation address is donating to this project
		"Ae2tdPwUPEZ68cfEjZjKKRabiqbazMtP69uGaM2pMZRg87fvn4FGvR95BEV").String()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	if len(c.Cryptocurrencies) != 3 {
		log.Println("lib/cryptocurrency supports new cryptocurrencies")
		log.Println("please update source code")
		return
	}

	defaultDests := map[c.Cryptocurrency]string{
		c.Bitcoin:  *donationAddressBTC,
		c.Ethereum: *donationAddressETH,
		c.Cardano:  *donationAddressADA,
	}

	db, err := database.Open(*databasePath)
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
		payHandler(db, client, ctx, w, r, defaultDests)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
