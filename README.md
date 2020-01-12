# donate

Cryptocurrency donation daemon.

## API

Query donation address for issue:

    curl -s 'https://donate.dumpstack.io/query?repo=github.com/jollheef/appvm&issue=3'

List all issues with cryptocurrency wallets (right now only BTC) address for repo:

    curl -s 'https://donate.dumpstack.io/query?repo=github.com/jollheef/appvm' | json_pp

Trigger payout:

    curl -s 'https://donate.dumpstack.io/pay?repo=github.com/jollheef/appvm&issue=3'

## Run locally (with [Nix](https://nixos.org/nix/))

    nix run -f https://code.dumpstack.io/tools/donate/archive/master.tar.gz -c donate

## Deploy

See [deploy/README.md](deploy/README.md).
