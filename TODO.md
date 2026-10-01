# TODO — nextendo-bcat-nx

## In progress

- **Icons inside an opened item of a custom channel** (next to the channel header, and on the related-channel
  buttons): not drawn, though qlaunch's article parser (0x65ba581e60) loads the top-level `topic_image` and each
  `related_channels[].topic_image` and the item opens. Not the size (70x70), not JFIF, not following the channel:
  a display condition in the article layout, not found yet. Nintendo's channels use built-in icons.
- **Opening a channel**: opens; each item's summary now matches qlaunch's parser (358x201 `list_image`, 70x70
  `icon_image`, `url`); confirm the entries show and open.
- **Nintendo's icon on `nx_news` / `nx_notice` items**: every item now carries our 70x70 `topic_image`; check
  whether it replaces the built-in icon there (it works for the custom channel).

- **News loading by itself**: the servers drive it, not the console app. Subscriptions: `titles/0100000000001000/
  topics` (HOME menu) lists the default channels and the console auto-subscribes them (status 3 seen for
  `nx_notice`). **Our addition:** `nx_news_nextendo` is in that list too, so every console auto-follows the
  Nextendo channel; untested whether the console accepts a non-Nintendo channel there: check with nextendo-nx
  Subscribe (subscribe.txt shows each channel's status) after a Clear news + reboot. Fetch-now: the penne push
  (baas-jwks TODO). Status 2 = subscribed (Follow), 3 = auto-subscribed.

- **Custom channels (`channels.json`)**: Diablo III is defined (`nx_news_diablo3`, game 01001b300b9be000) with a
  test item; confirm on a console: Find channels, its page, and whether the game's own News link opens it.

## Left

- **Game BCAT data** (in-game content): only some online games use BCAT (on this console: Smash Bros. Ultimate,
  Super Mario Bros. 35, Borderlands GOTY, Pokémon Legends Arceus, Marvel Ultimate Alliance 3 — not Diablo III).
  `list/nx_data_<title>` answers 400; to serve a game it needs that game's own passphrase (nextendo-nx Game data
  writes them to the SD card) and title id, and the files the game expects (BCAT-Toolbox `BcatList` format).
  BCAT passphrase (nextendo-nx → Diagnostics → Game data writes them to the SD card) and what files the game
  expects; the list format is BCAT-Toolbox's `BcatList`. Containers are encrypted with the game's own passphrase
  and title id, not the HOME menu's.
- **New items reach consoles only on their next scheduled check** (hours after a successful one): send the npns
  push that makes a console fetch now (production does), when a news file is added or changed.
- **`expired` channels**: lists always say `in_service`; add a way to retire a channel (`service_status: expired`)
  and find out what the console then does with its items.
- **Unknowns**: `shop` button `query` format; `movie` items; `mode`/`digest` values.
- **Remove** the old `/news` JSON and `news.json` once News works.

## Credits

- [CrustySean/BCAT-Toolbox](https://github.com/CrustySean/BCAT-Toolbox) (based on Random06457's BCAT-Manager) — News formats.
- The whole Nextendo Network team — https://nextendo.network. Nextendo is awesome.
