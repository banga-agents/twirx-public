# Documentation workspace

Mintlify uses `docs.json` as the current configuration format.

Preview locally:

```bash
npm install -g mint
mint dev
```

The CLI requires a supported Node.js version; use an LTS release. Documentation changes should also pass:

```bash
../scripts/check-docs.sh
```
