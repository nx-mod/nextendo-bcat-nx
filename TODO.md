# TODO — nextendo-bcat-nx

## In progress

- **Channel icons**: now 70x70 JPEGs (what qlaunch decodes); confirm on a console (Find channels, the custom channel).
- **Opening a channel fails**: `v2/topics/<channel>/online_archives` format unknown (same qlaunch code).

## Left

- **Game BCAT data**: `list/nx_data_<title>` still answers invented JSON; serve BCAT-Toolbox's `BcatList` format.
- **Per-game News channels**: `"game": "<title id>"` in a news file; Diablo III first.
- **New items reach consoles only on their next scheduled check** (hours after a successful one): send the npns
  push that makes a console fetch now (production does), when a news file is added or changed.
- **`expired` channels**: lists always say `in_service`; add a way to retire a channel (`service_status: expired`)
  and find out what the console then does with its items.
- **Unknowns**: `shop` button `query` format; `movie` items; `mode`/`digest` values.
- **Remove** the old `/news` JSON and `news.json` once News works.

## Credits

- [CrustySean/BCAT-Toolbox](https://github.com/CrustySean/BCAT-Toolbox) (based on Random06457's BCAT-Manager) — News formats.
- The whole Nextendo Network team — https://nextendo.network. Nextendo is awesome.
