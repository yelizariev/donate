// Copyright 2020 Mikhail Klementev. All rights reserved.
// Use of this source code is governed by a AGPLv3 license
// (or later) that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	c "code.dumpstack.io/lib/cryptocurrency"
	"code.dumpstack.io/tools/donate/database"
)

func getBtcBalance(btc string) (balance float64, err error) {
	urlf := "https://api.blockcypher.com/v1/btc/main/addrs/%s/balance"
	resp, err := http.Get(fmt.Sprintf(urlf, btc))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct{ Balance float64 }
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return
	}

	balance = result.Balance / 100000000
	return
}

func getEthBalance(address string) (balance float64, err error) {
	// TODO
	return
}

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

	body += "#### Current balance\n"
	for _, cc := range keys {
		wallet := issue.Wallets[cc]
		format := "- %.8f %s\n"
		var balance float64
		var err error
		switch cc {
		case c.Bitcoin:
			balance, err = getBtcBalance(wallet.Address)
			if err != nil {
				continue
			}
		case c.Ethereum:
			if wallet.Address == "" {
				continue
			}
			balance, err = getEthBalance(wallet.Address)
			if err != nil {
				continue
			}
		}
		symbol := strings.ToUpper(cc.Symbol())
		body += fmt.Sprintf(format, balance, symbol)
	}

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
