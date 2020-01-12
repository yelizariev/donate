# donate

Cryptocurrency donation daemon.

## API

Query donation address for issue:

    curl -s 'https://donate.dumpstack.io/query?repo=github.com/jollheef/appvm&issue=3'

List all issues with cryptocurrency wallets (right now only BTC) address for repo:

    curl -s 'https://donate.dumpstack.io/query?repo=github.com/jollheef/appm | json_pp

Trigger payout:

    curl -s 'https://donate.dumpstack.io/pay?repo=github.com/jollheef/appvm&issue=3'
