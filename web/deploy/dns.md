# DNS for twirx.org

**Registrar and DNS host:** Namecheap, BasicDNS (`dns1.registrar-servers.com`,
`dns2.registrar-servers.com`).

**Decision (2026-08-10):** keep DNS at Namecheap. Do **not** move nameservers to
Cloudflare. The apex is served directly from the project VPS using an `A`
record. IPv6 publication is deferred until an external client verifies the
route.

**Production status (2026-08-10):** `twirx.org` is live over HTTPS,
`www.twirx.org` redirects to the apex with HTTP 301, and the temporary port 8088
preview is closed.

The reason is email. Proton Mail is already fully configured on this zone, and a
nameserver migration means recreating every one of those records at the new
provider before cutting over. A missed record does not produce an error — it
produces mail that silently stops arriving.

---

## Records that already exist — do not touch

These are live and working. Nothing in the website deployment requires changing
any of them.

| Type | Host | Value |
|---|---|---|
| MX | `@` | `mail.protonmail.ch` (priority 10) |
| MX | `@` | `mailsec.protonmail.ch` (priority 20) |
| TXT | `@` | `v=spf1 include:_spf.protonmail.ch ~all` |
| TXT | `@` | `protonmail-verification=69f06821b5a2f9b1fcc262c2cd0f6ab15db0d372` |
| CNAME | `protonmail._domainkey` | `protonmail.domainkey.drwrik2q6q75ewsycmnowp63wpl673hb4k5vyozhsawvvrlerzovq.domains.proton.ch` |
| CNAME | `protonmail2._domainkey` | `protonmail2.domainkey.…proton.ch` |
| CNAME | `protonmail3._domainkey` | `protonmail3.domainkey.…proton.ch` |
| TXT | `_dmarc` | `v=DMARC1; p=quarantine` |

> If anyone ever proposes moving this zone, this table is the checklist that has
> to be reproduced first, and mail delivery has to be tested before the cutover.

---

## DNS records

Namecheap: **Domain List → Manage → Advanced DNS**. The *Host* field takes `@`
for the apex and a bare label for a subdomain (`docs`, not `docs.twirx.org`).

The former `CNAME www → parkingpage.namecheap.com` and `URL Redirect @ →
http://www.twirx.org/` parking records were removed before the cutover.

### 1. Documentation — `docs.twirx.org`

Independent of the main site. This can be done as soon as Mintlify shows you the
values.

| Step | Type | Host | Value |
|---|---|---|---|
| 1 | TXT | *(exactly as the Mintlify dashboard shows)* | *(exactly as shown)* |
| 2 | — | — | **Wait until Mintlify reports the domain verified** |
| 3 | CNAME | `docs` | *(the target the dashboard shows)* |

**Order matters.** Add the verification TXT first and wait for Mintlify to
confirm. Adding the CNAME before verification passes is the usual way this gets
stuck, and Mintlify's own guidance says the same.

**Use the target your dashboard displays, not one from a tutorial.** Mintlify has
changed infrastructure targets over time; a value copied from an older article
may be wrong for your account.

### 2. Main site — `twirx.org` and `www`

Only after the site is deployed and answers on the VPS preview. Pointing the
apex at a server that is not serving yet produces a broken domain, not a
placeholder.

| Type | Host | Value | Status |
|---|---|---|---|
| A | `@` | `116.202.50.220` | Active |
| CNAME | `www` | `twirx.org` | Active; Caddy redirects it to the canonical apex |
| AAAA | `@` | `2a01:04f8:0231:352a:0000:0000:0000:0002` | Deferred pending external IPv6 verification |

The build emits `_headers` and `_redirects` into `web/dist`, but Caddy does not
interpret them. The equivalent security headers, cache policy, and `www` → apex
redirect are in `web/deploy/twirx-production.caddy`.

### 3. Reserved names — create nothing

`proof`, `status`, `registry`, `api`, and `transparency` are reserved in the
architecture only. Public DNS should describe real operational surfaces, not
future ambition. Do not create empty records for them.

---

## CAA — later, deliberately

There is currently no CAA record, which is fine: with no CAA, any CA may issue.

Do **not** add one before both certificates exist. A CAA record that omits the CA
your host actually uses will block certificate issuance and renewal, and the
failure appears as an expired certificate months later. Once `twirx.org` and
`docs.twirx.org` are both serving valid HTTPS, check which CA issued each:

```bash
echo | openssl s_client -connect twirx.org:443 -servername twirx.org 2>/dev/null \
  | openssl x509 -noout -issuer
echo | openssl s_client -connect docs.twirx.org:443 -servername docs.twirx.org 2>/dev/null \
  | openssl x509 -noout -issuer
```

Then add a CAA that names **every** issuer in use, plus `iodef` for reports.

---

## Verification

After each change. DNS propagation on Namecheap BasicDNS is usually minutes, but
allow up to an hour before concluding something is wrong.

```bash
# Nameservers unchanged — this must still be Namecheap.
dig +short NS twirx.org

# Email must keep resolving. Run this after any zone edit.
dig +short MX twirx.org
dig +short TXT twirx.org
dig +short CNAME protonmail._domainkey.twirx.org

# Documentation
dig +short TXT docs.twirx.org
dig +short CNAME docs.twirx.org

# Main site
dig +short A twirx.org
dig +short AAAA twirx.org
dig +short CNAME www.twirx.org

# End to end, once serving
curl -sSI https://twirx.org | head -20
curl -sSI https://www.twirx.org | grep -i '^location'
curl -sS https://twirx.org/data/project-status.json | head -5
```

Confirm the security headers actually arrive — a CSP that is only in a file is
not a CSP:

```bash
curl -sSI https://twirx.org | grep -i 'content-security-policy'
```

And confirm the canonical URL matches the host being served, since the build
pins `canonical_host` to `twirx.org`:

```bash
curl -sS https://twirx.org/ | grep -o '<link rel="canonical"[^>]*>'
```

---

## Deployment order

1. Leave the already-live Mintlify `docs` records unchanged.
2. Build and verify `web/dist`, upload it as an immutable VPS release, and
   switch `/srv/twirx/site/current` atomically.
3. Activate `twirx-preview.caddy`; verify the site and headers at the VPS IP on
   port 8088.
4. Add the apex `A` record and the `www` CNAME above without
   changing any mail or documentation record.
5. Wait until both public names resolve to the VPS.
6. Replace the preview Caddy import with `twirx-production.caddy`; validate the
   complete Caddy configuration before reloading so automatic HTTPS can issue
   certificates.
7. Re-run the email checks above.
8. Verify headers, canonical URLs, the `www` redirect, and the published JSON
   endpoints.
9. Consider CAA once both certificates are issued.

Steps 1–8 were completed on 2026-08-10. The optional AAAA record remains
deferred.
