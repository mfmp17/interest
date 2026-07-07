# Wrong-chain / wrong-token recovery design

Deposit addresses are deterministic EVM EOAs. The same `0x...` address exists on Base, Polygon, Ethereum, Arbitrum, etc. If a user sends tokens on the wrong EVM chain, the backend can recover them **if** we run a watcher for that chain and the token is transferable.

## Current live watcher

Active today:

- Base mainnet
- Base native USDC: `0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913`

The Base watcher now classifies every ERC-20 `Transfer(..., to=depositAddress, ...)`:

- configured stable with matching decimals → accepted as principal
- unsupported token / mismatched decimals → admin alert stored on the position and logged as `ADMIN_ALERT`

## Polygon USDT example

To identify USDT accidentally sent on Polygon, add a second chain watcher with:

```bash
FRED_POLYGON_RPC_URL=<stable Polygon RPC>
FRED_POLYGON_USDT_ADDRESS=<verified Polygon USDT contract>
FRED_POLYGON_USDC_ADDRESS=<verified Polygon USDC contract>
```

The watcher should use the same deposit address and same derived private key index, but scan Polygon logs:

```txt
Transfer(address indexed from, address indexed to, uint256 value)
where to == depositAddress
```

If token matches Polygon USDT/USDC:

- record `wrong_chain_stable_detected`
- notify/log admin
- do NOT automatically count it into the Base position unless we intentionally support cross-chain crediting
- admin can export the same deposit private key and recover/sweep on Polygon

If token is not a configured stable:

- record `wrong_chain_unknown_token`
- notify/log admin
- export deposit private key only for genuine recovery

## Recovery key export

Admin-only endpoint:

```txt
GET /v1/admin/positions/:id/export-key
x-admin-token: <FRED_ADMIN_TOKEN>
```

Returns the private key for that deposit EOA. This is for operator recovery only; never expose it to users or the CLI.
