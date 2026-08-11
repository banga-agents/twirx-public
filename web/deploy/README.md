# Deploying the TWIRX website

The build output is plain files. There is no server runtime, no serverless
function, no database, and no origin API. Any static host, object store, or
ordinary web server can serve `web/dist` unchanged.

```bash
cd web
# Set base_url in site.json to the real domain first.
go run .
# Then publish web/dist by whatever means the host uses.
```

## Content Security Policy

The site loads nothing from any other host — no script, font, image, or
stylesheet — so it can run under a policy strict enough to be worth stating.
Serve these headers:

```text
Content-Security-Policy: default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'none'; connect-src 'none'; form-action 'none'; frame-ancestors 'none'; base-uri 'none'; object-src 'none'; upgrade-insecure-requests
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
Referrer-Policy: no-referrer
Cross-Origin-Opener-Policy: same-origin
Cross-Origin-Resource-Policy: same-origin
Permissions-Policy: accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=(), interest-cohort=()
```

Notes on the policy:

- `script-src 'self'` with no `'unsafe-inline'` and no `'unsafe-eval'`. Exactly
  one first-party module ships, on `/proof/explorer/` only; the build fails if a
  script appears on any other page, if it is inline, or if it references a remote
  URL or a dynamic import.
- `style-src 'self'` rather than `'unsafe-inline'`: there are no inline styles.
- `img-src 'self'` covers the SVG favicon. There are no other images.
- `connect-src 'none'`: no page fetches anything at runtime.
- `form-action 'none'`: there are no forms and no contact-form backend.
- Add `upgrade-insecure-requests` only once the site is served over HTTPS.

Verify after deployment that the browser console reports no CSP violation on
any of the sixteen pages. A violation means something was added that this project
said it would not ship.

## Caching

```text
/                     Cache-Control: public, max-age=300, must-revalidate
/*/                   Cache-Control: public, max-age=300, must-revalidate
/data/*.json          Cache-Control: public, max-age=300, must-revalidate
/styles.css           Cache-Control: public, max-age=3600
/icon.svg             Cache-Control: public, max-age=86400
```

Short lifetimes on HTML and on the JSON facts are deliberate: those files carry
status claims and funding totals that must be correctable quickly. The
stylesheet is not content-hashed, so it is kept to an hour rather than a year.

## VPS deployment with Caddy

The production VPS serves immutable releases from
`/srv/twirx/site/releases/<release-id>` through the atomic
`/srv/twirx/site/current` symlink. Build and verify locally, upload a new release,
then switch that symlink. Caddy serves only static output; Go is not required on
the server.

Two complete checked-in configurations are provided:

- [`twirx-preview.caddy`](twirx-preview.caddy) listens on port 8088 for
  pre-DNS verification and does not request certificates. Its CSP deliberately
  omits `upgrade-insecure-requests`, because that directive would make a browser
  request preview assets over unavailable HTTPS and render unstyled HTML.
- [`twirx-production.caddy`](twirx-production.caddy) serves the apex over
  automatic HTTPS and redirects `www` to it. Activate it only after both names
  resolve to the VPS.

The production shape is:

```caddyfile
twirx.org {
	tls rick@twirx.org
	root * /srv/twirx/site/current
	file_server
	encode zstd gzip

	header {
		Content-Security-Policy "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'none'; connect-src 'none'; form-action 'none'; frame-ancestors 'none'; base-uri 'none'; object-src 'none'; upgrade-insecure-requests"
		Strict-Transport-Security "max-age=31536000; includeSubDomains"
		X-Content-Type-Options "nosniff"
		Referrer-Policy "no-referrer"
		Cross-Origin-Opener-Policy "same-origin"
		Cross-Origin-Resource-Policy "same-origin"
		Permissions-Policy "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=(), interest-cohort=()"
		-Server
	}
}

www.twirx.org {
	tls rick@twirx.org
	header Strict-Transport-Security "max-age=31536000; includeSubDomains"
	redir https://twirx.org{uri} 301
}
```

### Atomic release permissions

The SSH deployment account is not a member of Caddy's private group. After an
upload, make the new release readable by Caddy **before** switching the active
symlink:

```bash
release=/srv/twirx/site/releases/<release-id>
sudo chgrp -R caddy "$release"
sudo find "$release" -type d -exec chmod 2750 {} +
sudo find "$release" -type f -exec chmod 0640 {} +
sudo -u caddy test -r "$release/index.html"
sudo -u caddy test -r "$release/styles.css"
```

The set-group-ID bit keeps the correct group on directories if a release is
inspected or amended before activation. A deployment must fail closed if the
two `sudo -u caddy test` checks fail. Keep the previous immutable release, and
switch `/srv/twirx/site/current` with a temporary symlink plus one atomic
`mv -T`; this makes rollback the same operation in reverse.

The checked-in files also apply the documented cache lifetimes and deny direct
access to the host-specific `_headers` and `_redirects` deployment metadata.
No access log is enabled for this site, so the server does not add an
undisclosed visitor log to a site that promises no analytics.

## Example: nginx

```nginx
server {
    root /srv/twirx/site/current;
    index index.html;

    add_header Content-Security-Policy "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; font-src 'none'; connect-src 'none'; form-action 'none'; frame-ancestors 'none'; base-uri 'none'; object-src 'none'; upgrade-insecure-requests" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer" always;
    add_header Cross-Origin-Opener-Policy "same-origin" always;
    add_header Cross-Origin-Resource-Policy "same-origin" always;
    add_header Permissions-Policy "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=(), interest-cohort=()" always;

    location / {
        try_files $uri $uri/ =404;
    }
}
```

## Hosts that read `_headers` and `_redirects`

The build now copies [`_headers`](_headers) and [`_redirects`](_redirects) into
the publish directory, because that is where Netlify and Cloudflare Pages read
them from. Between them they apply the CSP above and redirect `www.twirx.org`
and the host's own default hostname to the apex, so neither becomes a second
indexable origin. On a host that does not read these files they are inert text
and the headers must be configured as shown in the nginx or Caddy examples.

DNS for `twirx.org` is documented separately in [`dns.md`](dns.md), including
the Proton Mail records that must not be disturbed.

## Privacy properties to preserve

Whatever host is chosen, these must remain true, because the site states them
publicly in its own footer:

- no analytics, no tag manager, no pixel, no session recorder;
- no cookies set by the site;
- no external fonts;
- no third-party requests of any kind;
- no client-side JavaScript other than the single first-party explorer module,
  which the build keeps under an 8 KB gzipped budget and confines to
  `/proof/explorer/`.

If a host injects any of these automatically, the host is the wrong host.
Access logs kept by the server are outside the site's control; if they are
retained, say so publicly rather than claiming the site collects nothing.
