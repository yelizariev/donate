// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package database

import (
	c "code.dumpstack.io/lib/cryptocurrency"
)

// SeedPrivacy declare should be seed shown in wallet structure or not
type SeedPrivacy int

const (
	// HideSeed in wallet structure
	HideSeed SeedPrivacy = iota
	// Show in Wallet structure
	ShowSeed
)

// NewIssue returns issue with allocated wallets
func NewIssue() (issue Issue) {
	issue.Wallets = map[c.Cryptocurrency]Wallet{
		c.Bitcoin:  Wallet{},
		c.Ethereum: Wallet{},
	}
	return
}

// Issue with corresponding wallets for donations
type Issue struct {
	// Repo in format "github.com/jollheef/donate"
	Repo string
	// ID of the issue on GitHub (not internal database one!)
	ID int
	// Cryptocurrency wallets
	Wallets map[c.Cryptocurrency]Wallet
}

// Wallet for cryptocurrency
type Wallet struct {
	// Seed for wallet restoration
	Seed string `json:"-"`
	// Address for dotations
	Address string
}
