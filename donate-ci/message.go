// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"

	c "code.dumpstack.io/lib/cryptocurrency"
	"code.dumpstack.io/tools/donate/database"
	"github.com/google/go-github/v29/github"
)

func genBody(gh *github.Client, ctx context.Context, issue database.Issue) (
	body string, totalUSD float64) {

	body = "### Donate to this issue\n"

	var keys []c.Cryptocurrency
	for k, _ := range issue.Wallets {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		return int(keys[i]) < int(keys[j])
	})

	for _, cc := range keys {
		wallet := issue.Wallets[cc]
		var format string
		switch cc {
		case c.Bitcoin:
			format = "- BTC: [%s](https://blockchair.com/bitcoin/address/%s)\n"
		case c.Ethereum:
			if wallet.Address == "" {
				continue
			}
			format = "- ETH: [%s](https://etherscan.io/address/%s)\n"
		case c.Cardano:
			if wallet.Address == "" {
				continue
			}
			format = "- ADA: [%s](https://www.seiza.com/blockchain/address/%s)\n"
		}
		body += fmt.Sprintf(format, wallet.Address, wallet.Address)
	}

	body += "#### Current balance\n"
	for _, cc := range keys {
		wallet := issue.Wallets[cc]
		format := "- %.8f %s\n"
		balance, err := getBalance(cc, wallet.Address)
		if err != nil {
			continue
		}

		rate, err := getUSDConversionRate(cc)
		if err != nil {
			continue
		}

		totalUSD += balance * rate

		symbol := strings.ToUpper(cc.Symbol())
		body += fmt.Sprintf(format, balance, symbol)
	}

	body += fmt.Sprintf("- Total $%.2f\n", totalUSD)

	// > How to claim a bounty

	body += "\n<details><summary>How to claim a bounty</summary><p>\n\n"

	body += "1. Specify this issue in commit message ([keywords]" +
		"(https://help.github.com/en/github/managing-your-work-on-" +
		"github/closing-issues-using-keywords));\n"
	body += "2. Put to the body of pull request your"
	for _, cc := range c.Cryptocurrencies {
		body += " " + strings.ToUpper(cc.Symbol()) + ","
	}
	body = body[:len(body)-1]
	body += " addresses in the format:"
	for _, cc := range c.Cryptocurrencies {
		symbol := strings.ToUpper(cc.Symbol())
		body += fmt.Sprintf(" %s{your_%s_address},", symbol, cc.Symbol())
	}
	body = body[:len(body)-1] + ";"
	body += "\n3. When pull request will be accepted, you'll immediately " +
		"get all cryptocurrency to wallets that you're specified.\n"

	body += "\n</p></details>\n\n"

	// > Top 10 issues with a bounty (this repository)

	issues, err := getRepoIssues(issue.Repo)
	if err == nil && len(issues) != 0 {
		body += "<details><summary>" +
			"Top 10 issues with a bounty (this repository)" +
			"</summary><p>\n\n"

		body += dumpIssues(gh, ctx, issues)

		body += "\n</p></details>\n\n"
	}

	// > Top 10 issues with a bounty (all repositories)

	issues, err = getAllIssues()
	if err == nil && len(issues) != 0 {
		body += "<details><summary>" +
			"Top 10 issues with a bounty (all repositories)" +
			"</summary><p>\n\n"

		body += dumpIssues(gh, ctx, issues)

		body += "\n</p></details>\n\n"
	}

	// Footer

	body += "###### The default fee is 0% (someone who will solve this " +
		"issue will get all money without commission). " +
		"Consider donating to the [donation project]" +
		"(https://github.com/jollheef/donate) " +
		"itself, it'll help keep it work with zero fees. " +
		"[List of all issues with bounties](https://donate.dumpstack.io).\n"
	return
}

func dumpIssues(gh *github.Client, ctx context.Context, issues []issue) (s string) {
	for id, issue := range issues {
		fields := strings.Split(issue.URL, "/")
		if len(fields) != 5 {
			log.Println("url inside database is not valid")
			continue
		}
		owner := fields[1]
		repo := fields[2]
		no, err := strconv.Atoi(fields[4])
		if err != nil {
			log.Println("issue id is not valid")
			continue
		}

		redir := "https://donate.dumpstack.io/redirect?url=" + issue.URL

		ghIssue, _, err := gh.Issues.Get(ctx, owner, repo, no)
		var name string
		if err == nil {
			name = *ghIssue.Title
			name += " — "
		}

		url := fmt.Sprintf("%s[%s/%s#%d](%s)", name, owner, repo, no, redir)
		s += fmt.Sprintf("%d. %s — $%s\n", id+1, url, issue.USD)
	}
	return
}

func getUSDConversionRate(cc c.Cryptocurrency) (n float64, err error) {
	var id string
	switch cc {
	case c.Bitcoin:
		id = "bitcoin"
	case c.Ethereum:
		id = "ethereum"
	case c.Cardano:
		id = "cardano"
	default:
		errors.New(cc.Symbol() + " not supported")
	}

	format := "https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd"
	url := fmt.Sprintf(format, id)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct {
		Error string

		Bitcoin  struct{ USD float64 }
		Ethereum struct{ USD float64 }
		Cardano  struct{ USD float64 }
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return
	}

	if result.Error != "" {
		err = errors.New(result.Error)
		return
	}

	switch cc {
	case c.Bitcoin:
		n = result.Bitcoin.USD
	case c.Ethereum:
		n = result.Ethereum.USD
	case c.Cardano:
		n = result.Cardano.USD
	default:
		errors.New(cc.Symbol() + " not supported")
	}
	return
}

func getBalance(cc c.Cryptocurrency, address string) (n float64, err error) {
	switch cc {
	case c.Bitcoin, c.Ethereum:
		return getBalanceEthBtc(cc, address)
	case c.Cardano:
		return getBalanceAda(address)
	default:
		errors.New(cc.Symbol() + " not supported")
	}
	return
}

func getBalanceAda(address string) (n float64, err error) {
	payload := struct {
		Addresses []string `json:"addresses"`
	}{Addresses: []string{address}}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	body := bytes.NewReader(payloadBytes)

	url := "https://iohk-mainnet.yoroiwallet.com/api/txs/utxoSumForAddresses"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct{ Sum string }
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return
	}

	if result.Sum == "" {
		// not error, just zero
		return
	}

	sum, err := strconv.ParseInt(result.Sum, 10, 64)
	lovelaceFloat := new(big.Float).SetInt64(sum)
	oneADA := big.NewFloat(1000000)
	n, _ = new(big.Float).Quo(lovelaceFloat, oneADA).Float64()
	return
}

func getBalanceEthBtc(cc c.Cryptocurrency, address string) (n float64, err error) {
	format := "https://api.blockcypher.com/v1/%s/main/addrs/%s/balance"
	url := fmt.Sprintf(format, cc.Symbol(), address)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct {
		Error string
		// Balance in units
		Balance uint64
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return
	}

	if result.Error != "" {
		err = errors.New(result.Error)
		return
	}

	var oneUnit *big.Float
	switch cc {
	case c.Bitcoin:
		oneUnit = big.NewFloat(100000000)
	case c.Ethereum:
		oneUnit = big.NewFloat(1000000000000000000)
	default:
		errors.New(cc.Symbol() + " not supported")
	}

	units := new(big.Float).SetUint64(result.Balance)
	n, _ = new(big.Float).Quo(units, oneUnit).Float64()
	return
}
