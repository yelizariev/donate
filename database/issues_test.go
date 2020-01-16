// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package database

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	c "code.dumpstack.io/lib/cryptocurrency"
)

func TestDB(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp/", "donate_test_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := Open(filepath.Join(dir, "db.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}

	issue1 := Issue{
		Repo: "testrepo1",
		ID:   1,
		Wallets: map[c.Cryptocurrency]Wallet{
			c.Bitcoin: Wallet{
				Seed:    "btcSeed1",
				Address: "btcAddress1",
			},
			c.Ethereum: Wallet{
				Seed:    "ethSeed1",
				Address: "ethAddress1",
			},
		},
	}

	err = Add(db, issue1)
	if err != nil {
		t.Fatal(err)
	}

	issue2 := Issue{
		Repo: issue1.Repo,
		ID:   issue1.ID,
		Wallets: map[c.Cryptocurrency]Wallet{
			c.Bitcoin: Wallet{
				Seed:    "seed2",
				Address: "address2",
			},
			c.Ethereum: Wallet{
				Seed:    "seed2",
				Address: "address2",
			},
		},
	}

	err = Add(db, issue2)
	if err == nil {
		t.Fatal("UNIQUE constraint is not working")
	}

	exists, err := IsExists(db, issue1)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("issue is not exists")
	}

	issueThatNotExist := Issue{
		Repo: "11111",
		ID:   222222,
		Wallets: map[c.Cryptocurrency]Wallet{
			c.Bitcoin: Wallet{
				Seed:    "bitcoinSeedNotExists",
				Address: "bitcoinAddressNotExists",
			},
			c.Ethereum: Wallet{
				Seed:    "ethereumSeedNotExists",
				Address: "ethereumAddressNotExists",
			},
		},
	}

	exists, err = IsExists(db, issueThatNotExist)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("issue is exists")
	}

	issue4 := Issue{
		Repo: "repo4",
		ID:   4,
		Wallets: map[c.Cryptocurrency]Wallet{
			c.Bitcoin: Wallet{
				Seed:    "btcSeed4",
				Address: "btcAddress4",
			},
			c.Ethereum: Wallet{
				Seed:    "ethSeed4",
				Address: "ethAddress4",
			},
		},
	}

	err = Add(db, issue4)
	if err != nil {
		t.Fatal(err)
	}

	issue4after := NewIssue()
	issue4after.ID = issue4.ID
	issue4after.Repo = issue4.Repo

	err = GetWallets(db, &issue4after, ShowSeed)
	if err != nil {
		t.Fatal(err)
	}

	if len(issue4.Wallets) != len(issue4after.Wallets) {
		t.Fatal("invalid issue.Wallets")
	}
	if issue4.Repo != issue4after.Repo {
		t.Fatal("invalid issue: wrong repo")
	}

	if issue4.Wallets[c.Bitcoin].Seed != issue4after.Wallets[c.Bitcoin].Seed {
		t.Fatal("invalid issue: wrong wallet seed")
	}
	if issue4.Wallets[c.Bitcoin].Address != issue4after.Wallets[c.Bitcoin].Address {
		t.Fatal("invalid issue: wrong wallet address")
	}

	issue5 := Issue{
		Repo: "repo4",
		ID:   5,
		Wallets: map[c.Cryptocurrency]Wallet{
			c.Bitcoin: Wallet{
				Seed:    "btcSeed5",
				Address: "btcAddress5",
			},
			c.Ethereum: Wallet{
				Seed:    "ethSeed5",
				Address: "ethAddress5",
			},
		},
	}

	err = Add(db, issue5)
	if err != nil {
		t.Fatal(err)
	}

	issues, err := AllIssues(db, issue4.Repo, ShowSeed)
	if err != nil {
		t.Fatal(err)
	}

	if len(issues) != 2 {
		t.Fatal("invalid issues array")
	}

	if issues[0].Wallets[c.Bitcoin].Address != issue4.Wallets[c.Bitcoin].Address {
		t.Fatal("invalid issue: wrong wallet address")
	}

	if issues[1].Wallets[c.Ethereum].Address != issue5.Wallets[c.Ethereum].Address {
		t.Fatal("invalid issue: wrong wallet address")
	}

	if issues[1].Wallets[c.Bitcoin].Seed == "" {
		t.Fatal("seed is not shown")
	}

	if issues[1].Wallets[c.Ethereum].Seed == "" {
		t.Fatal("seed is not shown")
	}

	issues, err = AllIssues(db, issue4.Repo, HideSeed)
	if err != nil {
		t.Fatal(err)
	}

	if issues[1].Wallets[c.Bitcoin].Seed != "" {
		t.Fatal("seed is shown")
	}

	if issues[1].Wallets[c.Ethereum].Address == "" {
		t.Fatal("address is not shown")
	}
}
