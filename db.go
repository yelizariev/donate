// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import "database/sql"

func createIssuesTable(db *sql.DB) (err error) {
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS issues (
		id		INTEGER PRIMARY KEY,
		repo		TEXT NON NULL,
		issue		TEXT NON NULL,
		bitcoin_seed	TEXT NON NULL UNIQUE,
		bitcoin_address	TEXT NON NULL UNIQUE,
		UNIQUE(repo, issue) ON CONFLICT ROLLBACK
	)`)
	return
}

func createSchema(db *sql.DB) (err error) {
	err = createIssuesTable(db)
	if err != nil {
		return
	}

	return
}

func openDatabase(path string) (db *sql.DB, err error) {
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

func issueGet(db *sql.DB, repo, issue string) (seed, addr string, err error) {
	query := "SELECT bitcoin_seed, bitcoin_address " +
		"FROM issues " +
		"WHERE repo=? AND issue=?"
	stmt, err := db.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(repo, issue).Scan(&seed, &addr)
	return
}

func issueExists(db *sql.DB, repo, issue string) (exists bool, err error) {
	query := "SELECT EXISTS(" +
		"SELECT id FROM issues WHERE repo=? AND issue=?" +
		")"
	stmt, err := db.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(repo, issue).Scan(&exists)
	return
}

func issueAdd(db *sql.DB, repo, issue, seed, address string) (err error) {
	query := "INSERT INTO issues " +
		"(repo, issue, bitcoin_seed, bitcoin_address) " +
		"VALUES (?, ?, ?, ?)"
	stmt, err := db.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(repo, issue, seed, address)
	return
}

type issuePublicInfo struct {
	Issue, Address string
}

func issueAll(db *sql.DB, repo string) (issues []issuePublicInfo, err error) {
	query := "SELECT issue, bitcoin_address " +
		"FROM issues " +
		"WHERE repo = ?"
	stmt, err := db.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(repo)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		issue := issuePublicInfo{}
		err = rows.Scan(&issue.Issue, &issue.Address)
		if err != nil {
			return
		}

		issues = append(issues, issue)
	}
	return
}
