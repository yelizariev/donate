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
	tx, err := db.Begin()
	if err != nil {
		return
	}

	query := "SELECT id FROM issues WHERE repo=? AND issue=?"
	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()

	var issueID int
	err = stmt.QueryRow(repo, issue).Scan(&issueID)
	if err != nil {
		tx.Rollback()
		return
	}

	query = "SELECT seed, address FROM wallets " +
		"WHERE issue_id=? AND symbol='btc'"
	stmt, err = tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(issueID).Scan(&seed, &addr)
	if err != nil {
		tx.Rollback()
		return
	}

	tx.Commit()
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
	tx, err := db.Begin()
	if err != nil {
		return
	}

	query := "INSERT INTO issues (repo, issue) VALUES (?, ?)"
	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(repo, issue)
	if err != nil {
		tx.Rollback()
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return
	}

	query = "INSERT INTO wallets " +
		"(issue_id, symbol, seed, address) " +
		"VALUES (?, ?, ?, ?)"
	stmt, err = tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(id, "btc", seed, address)
	if err != nil {
		tx.Rollback()
		return
	}

	tx.Commit()
	return
}

type issuePublicInfo struct {
	Issue, Address string
}

func issueAll(db *sql.DB, repo string) (issues []issuePublicInfo, err error) {
	tx, err := db.Begin()
	if err != nil {
		return
	}

	query := "SELECT id, issue FROM issues WHERE repo = ?"
	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(repo)
	if err != nil {
		tx.Rollback()
		return
	}
	defer rows.Close()

	for rows.Next() {
		issue := issuePublicInfo{}
		var issueID int
		err = rows.Scan(&issueID, &issue.Issue)
		if err != nil {
			tx.Rollback()
			return
		}

		query = "SELECT address FROM wallets WHERE issue_id = ?"
		stmt, err = tx.Prepare(query)
		if err != nil {
			tx.Rollback()
			return
		}

		defer stmt.Close()

		err = stmt.QueryRow(issueID).Scan(&issue.Address)
		if err != nil {
			tx.Rollback()
			return
		}

		issues = append(issues, issue)
	}

	tx.Commit()
	return
}
