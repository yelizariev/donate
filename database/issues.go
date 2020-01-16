// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package database

import (
	"database/sql"

	c "code.dumpstack.io/lib/cryptocurrency"
)

// GetWallets for the issue. Repo and ID of the issue should be filled.
func GetWallets(db *sql.DB, issue *Issue, sp SeedPrivacy) (err error) {
	tx, err := db.Begin()
	if err != nil {
		tx.Rollback()
		return
	}
	err = txGetWallets(tx, issue, sp)
	if err != nil {
		tx.Rollback()
		return
	}
	return tx.Commit()
}

// same as GetWallets but to be wrapped by transaction
func txGetWallets(tx *sql.Tx, issue *Issue, sp SeedPrivacy) (err error) {
	id, err := getInternalID(tx, issue)
	if err != nil {
		return
	}

	query := "SELECT symbol, seed, address FROM wallets WHERE issue_id = ?"
	stmt, err := tx.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()

	rows, err := stmt.Query(id)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var symbol, seed, address string
		err = rows.Scan(&symbol, &seed, &address)
		if err != nil {
			return
		}

		var cc c.Cryptocurrency
		cc, err = c.FromSymbol(symbol)
		if err != nil {
			return
		}

		wallet := Wallet{Address: address}
		if sp == ShowSeed {
			wallet.Seed = seed
		}

		issue.Wallets[cc] = wallet
	}
	return
}

// getInternalID of the issue. Repo and ID of the issue should be filled.
func getInternalID(tx *sql.Tx, issue *Issue) (id int, err error) {
	query := "SELECT id FROM issues WHERE repo=? AND issue=?"
	stmt, err := tx.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(issue.Repo, issue.ID).Scan(&id)
	return
}

// IsExists check for the issue. Repo and ID of the issue should be filled.
func IsExists(db *sql.DB, issue Issue) (exists bool, err error) {
	query := "SELECT EXISTS(SELECT id FROM issues " +
		"WHERE repo=? AND issue=?)"
	stmt, err := db.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()

	err = stmt.QueryRow(issue.Repo, issue.ID).Scan(&exists)
	return
}

// Add issue to the database.
func Add(db *sql.DB, issue Issue) (err error) {
	tx, err := db.Begin()
	if err != nil {
		tx.Rollback()
		return
	}
	err = txAdd(tx, issue)
	if err != nil {
		tx.Rollback()
		return
	}
	return tx.Commit()
}

// same as Add but to be wrapped by transaction
func txAdd(tx *sql.Tx, issue Issue) (err error) {
	query := "INSERT INTO issues (repo, issue) VALUES (?, ?)"
	stmt, err := tx.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()

	res, err := stmt.Exec(issue.Repo, issue.ID)
	if err != nil {
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		return
	}

	for cc, wallet := range issue.Wallets {
		err = addWallet(tx, id, cc, wallet)
		if err != nil {
			return
		}
	}
	return
}

// addWallet to database by internal (database) issue ID
func addWallet(tx *sql.Tx, id int64, cc c.Cryptocurrency, wallet Wallet) (err error) {

	query := "INSERT INTO wallets " +
		"(issue_id, symbol, seed, address) " +
		"VALUES (?, ?, ?, ?)"
	stmt, err := tx.Prepare(query)
	if err != nil {
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(id, cc.Symbol(), wallet.Seed, wallet.Address)
	return
}

// AllIssues from database for repository
func AllIssues(db *sql.DB, repo string, sp SeedPrivacy) (issues []Issue, err error) {
	tx, err := db.Begin()
	if err != nil {
		tx.Rollback()
		return
	}
	issues, err = txAllIssues(tx, repo, sp)
	if err != nil {
		tx.Rollback()
		return
	}
	tx.Commit()
	return
}

// same as AllIssues but to be wrapped by transaction
func txAllIssues(tx *sql.Tx, repo string, sp SeedPrivacy) (issues []Issue, err error) {
	query := "SELECT issue FROM issues WHERE repo = ?"
	stmt, err := tx.Prepare(query)
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
		issue := NewIssue()
		issue.Repo = repo
		err = rows.Scan(&issue.ID)
		if err != nil {
			return
		}

		err = txGetWallets(tx, &issue, sp)
		if err != nil {
			return
		}

		issues = append(issues, issue)
	}
	return
}
