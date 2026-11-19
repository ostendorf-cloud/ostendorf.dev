# Page view counting

ostendorf.dev counts page views on the server with Umami, through govue's `analytics` package.
Nothing runs in the visitor's browser, and no request goes from the page to a third party. This
keeps the no-third-party-request property the site inherited from old_ostendorf.dev.

## What is counted

The statement below is the source for the privacy text. The privacy text itself belongs to the
legal pages work and is not written here. It follows govue's privacy checklist
(`docs/concepts/govue.md`, section "Analytics").

- A page view is counted on the server after the page was served. No script runs, no cookie is
  set and nothing is stored on the visitor's device.
- Only a `GET` that answered `200` with an HTML page counts. Assets (a directory under `/assets/`
  answers 404, never a listing), `robots.txt`, `sitemap.xml`, the 404 page, redirects, health and
  metrics are not pages and are not counted.
- The page is reported as its route pattern, for example `/`, never as the requested address. A
  query string, a token, an id or any other part of the address does not leave the server. A
  request that matched no route pattern counts nothing.
- What is sent to Umami: the route pattern, the referring site as an origin only (no path), the
  first language tag of the browser, the browser's user agent and the visitor's IP address.
- The IP address is shortened before it is sent: to its first three bytes for IPv4 and its first
  six bytes for IPv6. Umami derives country and usually region from it. The site never sets
  `FullIP`.
- A visitor who sends Do Not Track (`DNT: 1`) or Global Privacy Control (`Sec-GPC: 1`) is not
  counted at all.
- Prefetch requests, requests a browser marks as not a page navigation, bots, empty user agents
  and clients on private or loopback addresses are not counted.
- A slow or unreachable Umami never delays a response. A view that cannot be queued or sent is
  dropped and counted in `analytics_views_dropped_total`. A failure is logged at warn level: the
  first of a class at once, then at most once a minute with the count since the last line.

## What the owner still has to state

These are not properties of the code and belong in the privacy text:

- Who runs the Umami instance and where, and that no third party receives the data.
- What Umami stores of the shortened address, how long it keeps the data and the legal basis.

## Configuration

Counting is off unless both variables are set. Setting only one stops the start.

| Variable           | Meaning                                               | Secret |
| ------------------ | ----------------------------------------------------- | ------ |
| `UMAMI_URL`        | Base URL of the Umami instance, see below             | no     |
| `UMAMI_WEBSITE_ID` | UUID of the website in Umami                          | no     |
| `UMAMI_TIMEOUT`    | Bound of one post, 1s to 30s, default 5s              | no     |
| `UMAMI_QUEUE_SIZE` | Views that may wait for a worker, 1 to 10000, 256     | no     |
| `TRUSTED_PROXIES`  | CIDRs of the proxies in front, for the client address | no     |
| `SITE_URL`         | Public origin, its host is reported to Umami          | no     |

`UMAMI_URL` must be `https`. Plain `http` is accepted only for `localhost`, a loopback address
and a name only a cluster resolves (a single label such as `umami`, or a name ending in `.svc` or
`.svc.cluster.local`), because visitor addresses would otherwise travel in clear text. The URL
carries no credentials, query or fragment.

`UMAMI_TIMEOUT` and `UMAMI_QUEUE_SIZE` are read from the environment and listed in
`.env.example`. The Helm chart does not expose them: a deployment that needs them sets the
variables itself.
The start logs once whether counting is on, and warns when it is on with `TRUSTED_PROXIES` empty.
Behind an Ingress or a Gateway, `TRUSTED_PROXIES` must name the proxy range. Without it every
request comes from the proxy, which is a private address, and nothing is counted. In the Helm
chart the values are `config.umami.url`, `config.umami.websiteId` and `config.trustedProxyCidrs`.

## Wiring rule

`internal/webapp` wraps the mux directly with the analytics middleware. It reads the route
pattern the mux sets on the request, so no middleware that replaces the request, such as a
request id or security header middleware, may sit between the two. A page added to
`internal/routes/routes.json` is counted under its pattern with no further work, for example
`/en/` and `/de/` once those routes exist.
