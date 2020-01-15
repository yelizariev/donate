// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package database

import "database/sql"

func IssueGet(db *sql.DB, repo, issue string) (seed, addr string, err error) {
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

func IssueExists(db *sql.DB, repo, issue string) (exists bool, err error) {
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

func IssueAdd(db *sql.DB, repo, issue, seed, address string) (err error) {
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

func IssueAll(db *sql.DB, repo string) (issues []issuePublicInfo, err error) {
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
