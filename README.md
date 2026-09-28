# nextendo-bcat-nx

*(still in alpha testing)*

**A new service implementation by nx-mod** for the Nextendo Network.

BCAT (the Switch **d4c** delivery-cache service) for [Nextendo Network](https://nextendo.network), so a title's background content comes from the stack instead of Nintendo's CDN. Source only — no game data, no keys, no certs. Not affiliated with Nintendo.

## Why

BCAT is how Switch games pull background data — seasonal content, shared maps, config. A console fetches it over HTTPS from `bcat-list` / `bcat-topics` / `bcat-data` (`*.cdn.nintendo.net`). On the Nextendo stack these fall through to the real Nintendo CDN today, which is exactly the failure the game notes keep hitting: Borderlands' **"2122-0002 unable to load data"** and Advance Wars' `bcat-list` / `bcat-topics` (map share). This server answers them.

## How it works

- **Delivery cache from disk.** Content lives at `BCAT_CONTENT/<titleid-hex>/<directory>/<file>` (plus an optional `passphrase` file per title). The server scans it and serves an index (list) of directories, files, sizes and digests.
- **Signed containers.** Each data file is served wrapped in a BCAT content container (`bcat` magic, optional AES-CTR, RSA-2048 signature, PBKDF2-HMAC-SHA256 key derivation) — the format switchbrew documents.
- **Accepted by the console.** A stock console verifies every container against Nintendo's key. The `nextendo_bcat_sig` Atmosphère patch (bcat 22.5.0, installed by nextendo-nx) makes its signature check pass, so containers signed here are accepted. (Swapping the key itself, bcat-mitm, is not possible on 22.5.0: the key is not a static blob.)
- **News.** The HOME menu's News — channel catalog, channel lists, items — in the console's own formats. News items are one JSON file each: see **[NEWS.md](NEWS.md)**.

```
console ── HTTPS ──▶ sni-router ──▶ nextendo-bcat-nx ──▶ signed container
                                        ▲
                         stack/news/*.json (News items)
```

## Run

```sh
go build -o server .          # Go 1.24+ (uses crypto/pbkdf2). Stdlib only.
go test ./...
./server                      # generates a signing key on first run
./server pubkey               # print the public modulus for bcat-mitm
```

Put content under `BCAT_CONTENT`, point DNS/sni-router for the `bcat-*` hosts at this server, and install the matching public key in `bcat-mitm`. Config is in [example.env](example.env); every value is documented there.

| host (routed here) | endpoint | serves |
|---|---|---|
| `bcat-list-*` | `/list?title=<hex>` | the delivery-cache index (directories, files, digests) |
| `bcat-data-*` | `/data?title=<hex>&digest=<hex>` | one file as a signed container |
| `bcat-topics-*` | `/topics?title=<hex>` | topic set (empty by default) |
| — | `/api/stats?key=<DASH_TOKEN>`, `/healthz` | monitoring |

| port | what |
|---|---|
| 8470 | BCAT HTTP(S) (behind sni-router, or set `CERT_FILE`/`KEY_FILE` for direct TLS) |
| 8099 | `/api/stats`, `/healthz` |

## Status / known limits

Written from switchbrew (BCAT Content Container) and the d4c-emu / yuzu-Boxcat references; **not yet run against a console.** What is solid and tested here: the container format, PBKDF2 key derivation, AES-CTR, RSA-2048 sign/verify with the local key, the content scan and the list/data flow (8 tests). What still needs a capture to confirm (see [NOTES.md](NOTES.md)):

- the exact **file digest algorithm** (this uses SHA-256[:16]);
- the exact **RSA signed region and padding** (this uses RSA-PSS over header+payload) — this only has to match what a stock console recomputes;
- the real Nintendo **HTTP paths and the binary list index** layout (this serves a documented JSON index and accepts `/list`, `/data`, `/topics`);
- the per-title **passphrase / secretdata** values.

The companion **bcat-mitm** module (the on-console key replacement) is a separate project.

## Credits

- **[CrustySean/BCAT-Toolbox](https://github.com/CrustySean/BCAT-Toolbox)** (based on **[Random06457](https://github.com/Random06457)**'s BCAT-Manager) — the News formats: catalog, channel lists, news records, container layout.
- **[kinnay/NintendoClients wiki](https://github.com/kinnay/NintendoClients/wiki/BCAT-Servers)** — the BCAT server endpoints.
- **[switchbrew](https://switchbrew.org/wiki/BCAT_services)** — the BCAT service and content-container documentation.
- **[D3fau4/d4c-emu](https://github.com/D3fau4/d4c-emu)** and **yuzu's Boxcat** — BCAT server references.

Formats were read and reimplemented; no code was copied.

Built by nx-mod for the **Nextendo Network**, on the work of the whole Nextendo Network team — https://nextendo.network. Nextendo is awesome.
