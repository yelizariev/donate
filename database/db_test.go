// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package database

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestDB(t *testing.T) {
	dir, err := ioutil.TempDir("/tmp/", "donate_test_")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := openDatabase(filepath.Join(dir, "db.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}

	type testData struct{ Repo, Issue, Seed, Address string }

	test := testData{
		Repo:    "testrepo",
		Issue:   "testissue",
		Seed:    "testseed",
		Address: "testaddress",
	}

	err = issueAdd(db, test.Repo, test.Issue, test.Seed, test.Address)
	if err != nil {
		t.Fatal(err)
	}

	err = issueAdd(db, test.Repo, test.Issue, "otherseed", "otheraddress")
	if err == nil {
		t.Fatal("UNIQUE constraint is not working")
	}

	exists, err := issueExists(db, test.Repo, test.Issue)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("issue is not exists")
	}

	exists, err = issueExists(db, "not exists for sure", "100500")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("issue is exists")
	}

	test2 := testData{
		Repo:    test.Repo,
		Issue:   "test2issue",
		Seed:    "test2seed",
		Address: "test2address",
	}

	err = issueAdd(db, test2.Repo, test2.Issue, test2.Seed, test2.Address)
	if err != nil {
		t.Fatal(err)
	}

	seed, addr, err := issueGet(db, test.Repo, test.Issue)
	if err != nil {
		t.Fatal(err)
	}
	if test.Seed != seed || test.Address != addr {
		t.Fatal("invalid issue")
	}

	issues, err := issueAll(db, test.Repo)
	if err != nil {
		t.Fatal(err)
	}

	if len(issues) != 2 {
		t.Fatal("invalid issues array")
	}

	if issues[0].Issue != test2.Issue ||
		issues[1].Issue != test.Issue ||
		issues[0].Address != test2.Address ||
		issues[1].Address != test.Address {

		t.Fatal("invalid issues array")
	}
}
