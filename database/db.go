// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// Open (or create new) sqlite3 database on the path,
// and generate all tables if they do not exist.
func Open(path string) (db *sql.DB, err error) {
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		return
	}

	err = createSchema(db)
	if err != nil {
		return
	}

	return
}

func createSchema(db *sql.DB) (err error) {
	err = createIssuesTable(db)
	if err != nil {
		return
	}

	err = createWalletsTable(db)
	if err != nil {
		return
	}

	return
}

func createIssuesTable(db *sql.DB) (err error) {
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS issues (
		id		INTEGER PRIMARY KEY,
		repo		TEXT NON NULL,
		issue		TEXT NON NULL,
		UNIQUE(repo, issue) ON CONFLICT ROLLBACK
	)`)
	return
}

func createWalletsTable(db *sql.DB) (err error) {
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS wallets (
		id		INTEGER PRIMARY KEY,
		issue_id	INTEGER,
		symbol		TEXT NON NULL,
		seed		TEXT NON NULL UNIQUE,
		address		TEXT NON NULL UNIQUE
	)`)
	return
}
