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
  `nx_notice`); confirm after a Clear + reboot, and decide whether `nx_news_nextendo` joins that list. Fetch-now:
  the penne push (baas-jwks TODO). Status 2 = subscribed (Follow), 3 = auto-subscribed.

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
