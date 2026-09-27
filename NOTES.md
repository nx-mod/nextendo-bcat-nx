# nextendo-bcat-nx — notes

Keep this file updated as you go. Everything here is either from public
documentation (switchbrew, d4c-emu, yuzu Boxcat) or marked **UNCONFIRMED**
(needs a capture or a console).

## What BCAT is

Nintendo's background content delivery ("d4c"). A console fetches a title's
delivery cache over HTTPS from three hosts:

- `bcat-list-*.cdn.nintendo.net`   the index of directories/files (+ digests)
- `bcat-data-*.cdn.nintendo.net`   each file's data, addressed by digest
- `bcat-topics-*.cdn.nintendo.net` topic metadata

`nn::bcat` (IBcatService / IDeliveryCacheStorageService / IDeliveryCacheProgressService)
drives it on the console side. Games seen using it on this stack: Borderlands
(delivery cache; "2122-0002 unable to load data" when it fails), Advance Wars
(map share; `bcat-list`/`bcat-topics`).

## Content container (switchbrew: BCAT Content Container) — implemented

0x120-byte header, then the (optionally AES-CTR) payload:

```
0x00 [4]    magic "bcat"
0x05 u8     crypto: 1=AES-128-CTR 2=AES-192 3=AES-256 else=plaintext
0x06 u8     hash:   0,2=SHA-1  1,3=SHA-256
0x07 u8     secretdata index
0x10 [0x10] IV/CTR
0x20 [0x100] RSA-2048 signature
0x120 ...   payload
```

Key derivation: PBKDF2-HMAC-SHA256, 4096 iterations; password = the title
passphrase (<=0x40); salt = `"%016x%s"` over (title id, secretdata). All of this
is in `container.go` and covered by tests.

## The Nintendo-key replacement (the module's job)

The 0x100 signature is checked by a stock console against Nintendo's RSA-2048
public key in `nn::bcat`. To serve our own content we sign with a **local**
RSA-2048 key (`keys.go`, `BCAT_KEY_FILE`) and the on-console **bcat-mitm** module
patches Nintendo's public key to ours, so the console accepts it. `server pubkey`
prints the modulus; `<key>.pub.der` is written for the module to embed. The
private key never leaves the server.

## UNCONFIRMED — needs a capture / console

1. **File digest algorithm.** switchbrew's DeliveryCacheFileMetaEntry has a 0x10
   digest; the algorithm over the data is not published. We use SHA-256[:16]. A
   stock console recomputes the digest of fetched data and compares — so this
   must match the console's algorithm. Capture a real `bcat-list` + `bcat-data`
   pair and check.
2. **Signed region + padding.** We sign RSA-PSS/SHA-256 over (header minus the
   signature field) + payload. Nintendo's exact region and padding (PSS vs
   PKCS#1v1.5) are unknown; whatever it is, `bcat-mitm` and this server must
   agree, and it must be what `nn::bcat` verifies. Confirm from the binary.
3. **HTTP paths + list index.** The real request paths (and whether the index is
   the JSON we serve or a binary blob) are not published. We route by Host
   prefix and accept `/list`, `/data`, `/topics` with `?title=`/`?digest=`.
   Capture the console's requests (the bl1-hack/aw-hack BCAT hooks already log
   MountDeliveryCacheStorage / RequestSyncDeliveryCache / DeliveryCacheFile
   Open/Read — pair that with a network capture).
4. **Passphrase / secretdata.** Per-title values; put a `passphrase` file in the
   title's content dir, or set `BCAT_PASSPHRASE`. Real values come from the game.

## How to close the gaps

1. Route the three `bcat-*` hosts to this server (sni-router + hosts), install
   the local public key in `bcat-mitm`, and drop a small delivery cache under
   `BCAT_CONTENT` for one game (Advance Wars map share is the smallest target).
2. Capture the console's BCAT HTTP with the game's logger (bl1-hack/aw-hack BCAT
   hooks + a TLS/network capture) to settle 1–4 above.
3. Adjust `fileDigest`, `signContainer`/`signedRegion`, and the handlers to match,
   then the delivery lands end to end.

## Ports

| what | port |
|---|---|
| BCAT HTTP(S) (behind sni-router) | 8470 |
| dashboard | 8099 |
