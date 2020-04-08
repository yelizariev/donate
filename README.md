![GitHub Actions](https://github.com/jollheef/donate/workflows/Build/badge.svg)
[![Donate](https://img.shields.io/badge/donate-paypal-blue.svg)](https://www.paypal.com/cgi-bin/webscr?cmd=_s-xclick&hosted_button_id=R8W2UQPZ5X5JE&source=url)
[![Donate](https://img.shields.io/badge/donate-bitcoin-blue.svg)](https://blockchair.com/bitcoin/address/bc1q23fyuq7kmngrgqgp6yq9hk8a5q460f39m8nv87)

# donate

Cryptocurrency donation daemon.

Goals:
- KISS.
- Zero-fee (the fee is voluntary as a donation to the project).
- Self-hosted.
- Multiple cryptocurrencies (Bitcoin, Ethereum and Cardano).
- Multiple hosting (so far GitHub only).

How it works:

0. (optional) The owner of the repository does setting up a donation daemon.
1. The owner of the repository adds [GitHub action](.github/workflows/donate.yml) (it's the easiest way to work with GitHub).
2. Someone opens an issue, then GitHub action shows cryptocurrency addresses (and updates from time to time).
3. Someone solves the issue, adds to commit message `Fixes #N`, then put to pull request his BTC, ETH, ADA addresses in the format: BTC{address}, ETH{address}, ADA{address} et cetera;
4. The pull request is accepted
5. Repository administrator adds label "Paid" to the issue
6. GitHub Action triggers payout on donation daemon.
7. If no one acquired money then payout going to donation address (default is donating to this project).

This project uses [Semantic Versioning](https://semver.org/).

## API

Query donation address for issue:

    curl -s 'https://donate.dumpstack.io/query?repo=github.com/jollheef/appvm&issue=3'

List all issues with cryptocurrency wallets address for repo:

    curl -s 'https://donate.dumpstack.io/query?repo=github.com/jollheef/appvm' | json_pp

Trigger payout:

    curl -s 'https://donate.dumpstack.io/pay?repo=github.com/jollheef/appvm&issue=3'

## Run locally (with [Nix](https://nixos.org/nix/))

    nix run -f https://code.dumpstack.io/tools/donate/archive/master.tar.gz -c donate

## Deploy

See [deploy/README.md](deploy/README.md).
