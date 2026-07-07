# Fred Backend Deployment

Persistent backend for custodial USDC deposits, locks, claims, sweeps, and withdrawals.

## Current production defaults

- Network: Base mainnet
- Chain ID: `8453`
- Token: Base native USDC `0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913`
- Lock: `31536000` seconds (365 days)
- State: JSON file at `FRED_STATE_PATH`

## Required production env vars

Set these on the host:

```bash
FRED_RPC_URL=<stable Base mainnet RPC URL>
FRED_MASTER_SEED_HEX=<hex seed for deposit-address derivation>
FRED_TREASURY_PRIVATE_KEY=<hot wallet private key, no 0x or with 0x both accepted>
FRED_STATE_PATH=/data/mainnet-state.json
FRED_LOCK_SECONDS=31536000
FRED_CHAIN_ID=8453
FRED_NETWORK=base-mainnet
FRED_CONFIRMATIONS=4
```

For a tiny mainnet canary, temporarily use:

```bash
FRED_LOCK_SECONDS=365
```

Then switch back to `31536000` after the canary.

## Railway notes

Deploy from this `backend/` directory so Railway sees the Dockerfile:

```bash
cd ~/fred/backend
railway init
railway up
```

Add a Railway volume mounted at:

```txt
/data
```

Set:

```txt
FRED_STATE_PATH=/data/mainnet-state.json
```

Do not use ephemeral container filesystem for production state. The receipt token lives client-side; the position/lock state lives here. Losing state means losing the backend's record of anonymous deposits.

## Health/status

```bash
curl https://<backend-host>/health
curl https://<backend-host>/v1/status
```

## Mainnet canary flow

1. Deploy with `FRED_LOCK_SECONDS=365`.
2. Fund treasury with a small amount of Base ETH + USDC.
3. Run:

```bash
INTEREST_API=https://<backend-host> interest deposit
```

Use `$10 USDC`, Instant 5%.

4. Send exactly 10 USDC on Base mainnet to the shown deposit address.
5. Run:

```bash
INTEREST_API=https://<backend-host> interest balance
INTEREST_API=https://<backend-host> interest withdraw
```

6. After 365 seconds, run `interest withdraw` again to test principal withdrawal.
7. If clean, set `FRED_LOCK_SECONDS=31536000`, redeploy/restart, then publish CLI v0.2.0.
