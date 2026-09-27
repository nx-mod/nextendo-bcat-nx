# nextendo-bcat-nx — example usage

Runs from this testing branch with no setup: `./run.sh` (uses the shipped
`bcat_signing_key.pem`, `content/`, `news.json`).

## Serve news to the console

The Switch News/topics sync pulls from the BCAT topics host, redirected here by
the stack. This server serves a configurable news feed — by default the single
item **"nextendo news test"**.

```sh
./run.sh &                 # starts on :8470 (dash :8110)
./test-news.sh             # -> {"news":[{"message":"nextendo news test",...}]}
```

Change the headline by editing `news.json`:

```json
[ { "id": "welcome", "title": "Nextendo", "message": "Welcome to Nextendo!" } ]
```

## Serve a delivery-cache container (Splatoon/SMB35 event data)

Put a title's files under `content/<titleid>/` and the server signs them into
BCAT containers on request (`/list`, `/data`). The matching on-console module
`nextendo-bcat-mitm-nx` makes the console trust this server's signing key.

```sh
curl -s "http://localhost:8470/list?title=<titleid>"
```

> The News applet's exact on-console container schema is not fully published;
> this serves the tractable JSON + signed containers and logs requests. See
> NOTES.md.
