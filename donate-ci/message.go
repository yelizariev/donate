// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sort"
	"strings"

	c "code.dumpstack.io/lib/cryptocurrency"
	"code.dumpstack.io/tools/donate/database"
)

func genBody(issue database.Issue) (body string) {
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
		}
		body += fmt.Sprintf(format, wallet.Address, wallet.Address)
	}

	var totalUSD float64

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

	body += "\nUsage:\n"
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
	body += "###### The default fee is 0% (someone who will solve this " +
		"issue will get all money without commission). " +
		"Consider donating to the [donation project]" +
		"(https://github.com/jollheef/donate) " +
		"itself, it'll help keep it work with zero fees.\n"
	return
}

func getUSDConversionRate(cc c.Cryptocurrency) (n float64, err error) {
	var id string
	switch cc {
	case c.Bitcoin:
		id = "bitcoin"
	case c.Ethereum:
		id = "ethereum"
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
	default:
		errors.New(cc.Symbol() + " not supported")
	}
	return
}

func getBalance(cc c.Cryptocurrency, address string) (n float64, err error) {
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
