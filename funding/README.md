# Funding records

This directory contains public, machine-readable treasury records.

| File or directory | Purpose |
|---|---|
| `wallets.json` | Declared public project wallets and control status |
| `ledger.csv` | Every declared project inflow and outflow |
| `maintainer-runway.md` | Compensation disclosure policy and period records |
| `expenses/` | Human-readable expense records |
| `receipts/` | Redacted public evidence and original-document hashes |

## Never commit

- seed phrases;
- private keys;
- wallet backup files;
- exchange API keys;
- identity documents;
- personal addresses;
- unredacted payment card or bank details;
- invoices containing sensitive personal data.

The ignored path `funding/private/` is not a secure vault. It is only an accidental-commit guard. Keep secrets outside the repository entirely.
