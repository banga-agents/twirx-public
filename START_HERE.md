# Start here

This bundle is a runnable Genesis bootstrap plus the first scoped Codex work order.

## 1. Establish the local repository

```bash
unzip typed-web-genesis.zip
cd typed-web-genesis
git init -b main
git add .
git commit -m "genesis: establish evidence-native source-statement slice"
```

The module path is provisionally:

```text
github.com/typed-web-commons/typed-web
```

Change it before the public repository is created if the organization or project name differs:

```bash
go mod edit -module github.com/YOUR_ORG/YOUR_REPO
# Update the matching import prefix throughout cmd/ and internal/.
gofmt -w cmd internal
go test ./...
```

## 2. Validate the bootstrap

```bash
make clean
make build
make test
make demo
```

Do not proceed if the independent C verifier, corrupted-evidence rejection, offline replay, or documentation checks fail.

## 3. Give Codex the first work order

At the repository root, instruct Codex:

```text
Read AGENTS.md and tasks/001-genesis-source-statement-slice.md. Execute that work order exactly. Begin by running the required baseline commands. Do not widen scope or add dependencies. Preserve all constitutional and security invariants. Finish with the required Gate 1 evidence report.
```

The task assumes the existing bootstrap is pre-alpha code that must be audited, not trusted by default.

## 4. Create the GitHub repository

After Gate 1 passes locally:

```bash
gh repo create typed-web --public --source=. --remote=origin --push
```

Adjust the owner and repository name deliberately. Then enable:

- private vulnerability reporting;
- branch protection for `main`;
- required CI checks;
- secret scanning;
- Dependabot or an equivalent dependency alert process;
- discussion and issue templates only after contribution workflows are ready.

## 5. Connect Mintlify

Use the `docs/` directory as the documentation source. Mintlify's current configuration file is `docs/docs.json`.

Local preview:

```bash
cd docs
npm install -g mint
mint dev
```

The repository includes `docs/.mintlify/AGENTS.md` so documentation agents preserve protocol terminology and implemented/future boundaries.

## 6. Publish the technical website

The `site/` directory is dependency-free static HTML and CSS.

```bash
cd site
python3 -m http.server 8080
```

Deploy it only after replacing provisional project references and adding the real repository and documentation URLs.

## 7. Declare funding safely

Before sharing a wallet:

1. Create a dedicated project wallet separate from personal trading wallets.
2. Back up its recovery material offline.
3. Never paste its seed phrase or private key into Codex, ChatGPT, GitHub, cloud notes, repository secrets, or funding files.
4. Add only the public address and control description to `funding/wallets.json`.
5. Commit that declaration before accepting funds.
6. Record every inflow and outflow in `funding/ledger.csv`.

Read `FUNDING.md`, `TREASURY.md`, and `funding/README.md` first.
