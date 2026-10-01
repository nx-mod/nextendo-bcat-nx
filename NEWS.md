# Adding news

bcat-nx serves the HOME menu's **News** the way the console fetches it on its own (on boot, on login, and
from News itself). Each news item is **one JSON file** in the news folder; add, edit or delete files and
the next fetch picks them up. No restart, no build.

- On the nextendo-testing stack the folder is `stack/news/` (set by `BCAT_NEWS_DIR`).
- The console needs the `nextendo_bcat_sig` patch (see below), installed by nextendo-nx.

## Add an item

Create `stack/news/<name>.json`:

```json
{
  "title": "Servers are back",
  "body": "Maintenance is over. Have fun!",
  "date": "2026-09-28"
}
```

That's all a basic item needs. To show a picture, put a JPEG next to it with the same name
(`stack/news/<name>.jpg`) — it becomes the thumbnail in the News list.

The console picks new items up on its own schedule (hours after its last check); a push that makes it check at once
is not done yet (see TODO.md).

## Fields

| field | what | default |
|---|---|---|
| `title` | headline | the file name |
| `body` | the text | — |
| `footer` | small text under the body | none |
| `date` | `YYYY-MM-DD` or `2026-09-28T10:00:00Z`; newest shows first | the file's time |
| `channel` | `news` (main feed, everyone sees it), `notice` (notices), `nextendo` (the Nextendo channel, found under **Find channels**) | `news` |
| `image` | another image file in the folder to use | `<name>.jpg` if it exists, else `default.jpg` (the logo) |
| `picture` | `true` also shows the image full size in the body | `false` |
| `button` | a button under the item (below) | none |
| `movie` | a video URL in the body (untested) | none |
| `featured` | `true` puts it on the lock screen and in the featured row; only the 3 newest featured items stay featured | `false` |
| `related` | other channels shown under **Related channels** in the opened item, e.g. `["nextendo", "notice"]` (its own channel is always there) | none |
| `priority` | advanced: 1000 and up is featured (highest first); anything lower is the normal list | `1500` if featured, else `100` |
| `id` | fixed news id (number) | derived from the file name |
| `extra` | raw fields added to the record as-is, for anything not covered above (see the format below) | none |

Channel icons: `<channel>-icon.jpg` or `icon.jpg` in the folder, else `default.jpg`; served as a 70x70 JPEG (the only
size the console shows). Nintendo's own channels (`news`, `notice`) keep their built-in icons.

Images: JPEG, up to 1 MB. The console's own items use a wide thumbnail; any size shows.

### Buttons

```json
"button": { "type": "browser",  "text": "Open website", "url": "https://nextendo.network" }
"button": { "type": "game",     "text": "Play",         "title_id": "01001B300B9BE000" }
"button": { "type": "settings", "text": "News settings", "applet": 4 }
"button": { "type": "shop",     "text": "See in eShop", "query": "..." }
```

`settings` opens a System Settings page: `applet` 4 is News settings, 2 Parental Controls. `shop` is
untested: its `query` format is not known yet.

`stack/news/` has one example of each type (`news-test`, `notice-test`, `image-test`, `browser-button-test`,
`game-button-test`, `settings-button-test`, `channel-test`). Delete them when you add real news.

## Check it

bcat-nx logs each fetch (`stack/logs/bcat-nx.log`):

    [BCAT news] list nx_news -> 5 item(s)
    [BCAT] GET bcat-data-lp1.cdn.nintendo.net/api/nx/v1/news/nx_news/734... -> 200

A file that isn't valid JSON is skipped and logged. To see what a console receives without one:

    curl -k --resolve bcat-list-lp1.cdn.nintendo.net:443:<stack IP> https://bcat-list-lp1.cdn.nintendo.net/api/nx/v1/list/nx_news

## How it works

The console asks, over HTTPS to the stack (sni-router → bcat-nx):

| request | reply |
|---|---|
| `bcat-topics …/api/nx/v1/titles/0100000000001000/topics` | the channels every console follows by default |
| `bcat-topics …/api/nx/v1/topics/catalog` | the channels (Find channels) |
| `bcat-topics …/api/nx/v1/topics/<channel>/detail`, `/icon` | one channel; the icon is a 70x70 JPEG |
| `bcat-topics …/api/nx/v2/topics/<channel>/online_archives` | a channel's page: its items, each with a summary URL |
| `bcat-data …/api/nx/v1/news/<channel>/<id>/summary` | an item as the channel's page lists it (358x201 thumbnail) |
| `bcat-list …/api/nx/v1/list/<channel>` | the channel's items: ids, URLs, sizes |
| `bcat-data …/api/nx/v1/news/<channel>/<id>` | one item: the news record |

**Default channels.** `news` and `notice` are Nintendo's defaults. `nextendo` is **our addition** to that list, so
every console follows the Nextendo channel without the user finding it; production's list has only Nintendo's.

**When items arrive.** A console fetches a channel's list when it subscribes to it (Follow) and then on its own
schedule, hours apart. Making it fetch at once needs the push server (penne), which is not done yet.

Every reply is a BCAT content container: MessagePack inside, AES-128-CTR encrypted with the HOME menu's News
passphrase, signed with the stack's BCAT key. A stock console only accepts Nintendo's signature; the
`nextendo_bcat_sig` Atmosphère patch (`atmosphere/exefs_patches/nextendo_bcat_sig/`, for bcat on 22.5.0) makes it
accept the stack's.

The formats are the console's own: the record from the local notices dumped with nextendo-nx's **Dump news**, the
rest read from the bcat and HOME menu code of firmware 22.5.0 (sizes and required values are firmware-specific).

## Credits

- [CrustySean/BCAT-Toolbox](https://github.com/CrustySean/BCAT-Toolbox), based on [Random06457](https://github.com/Random06457)'s BCAT-Manager — News formats.
- [kinnay/NintendoClients wiki](https://github.com/kinnay/NintendoClients/wiki/BCAT-Servers) — BCAT endpoints.
- The whole Nextendo Network team — https://nextendo.network. Nextendo is awesome.
