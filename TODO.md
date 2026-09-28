# TODO — nextendo-bcat-nx

## In progress

- **Confirm News on a console**: Find channels, the feed, each item (encrypted containers). The console's
  `nextendo_bcat_sig` now covers every signature check; if it still fails, suspect the catalog format.

## Left

- **Game BCAT data**: `list/nx_data_<title>` still answers invented JSON; serve BCAT-Toolbox's `BcatList` format.
- **Per-game News channels**: `"game": "<title id>"` in a news file; Diablo III first.
- **Unknowns**: `shop` button `query` format; `movie` items; `mode`/`digest` values.
- **Remove** the old `/news` JSON and `news.json` once News works.

## Credits

- [CrustySean/BCAT-Toolbox](https://github.com/CrustySean/BCAT-Toolbox) (based on Random06457's BCAT-Manager) — News formats.
- The whole Nextendo Network team — https://nextendo.network. Nextendo is awesome.
