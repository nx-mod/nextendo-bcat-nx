# Running nextendo-bcat-nx (prebuilt binary)

These are prebuilt Linux binaries of **nextendo-bcat-nx** for a home-lab LAN. No Go toolchain
needed.

## Quick start

1. Download the binary for your CPU from this `dist/` folder:
   - `nextendo-bcat-nx-linux-amd64`  (Intel/AMD, most PCs and VPS)
   - `nextendo-bcat-nx-linux-arm64`  (Raspberry Pi 4/5, ARM servers)
2. Put it in a folder together with this repo's runtime files (the config that
   ships on this `testing` branch — e.g. `cert.pem`, `key.pem`, `*.json`,
   signing keys). The server reads them from its working directory.
3. Make it executable and run it:

```sh
chmod +x nextendo-bcat-nx-linux-amd64
./nextendo-bcat-nx-linux-amd64
```

Or, from a clone of this branch, just `./run.sh` (it sets sane env and runs the
binary). Dashboard/stats: `http://<host>:8110`.

> Test build off the `testing` branch. TLS is normally fronted by the
> sni-router; the shipped self-signed cert is accepted because Prelude/nextendo-nx
> installs `disable_ca_verification`.
