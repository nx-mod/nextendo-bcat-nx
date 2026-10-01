# Fixed — nextendo-bcat-nx

- **News "Find channels" TLS refused**: its own certificate covering `*.cdn.nintendo.net` (BCAT checks the name).
- **Containers rejected**: header version byte set to 1, as bcat requires.
- **Requests never logged**: every request logged with its status.
- **News**: catalog, channels, lists and items in the console's formats, encrypted; items are JSON files (NEWS.md).
- **News lists ignored (no item ever downloaded)**: `service_status` is `in_service`; bcat 22.5.0 rejects anything but
  `in_service`/`expired` (0x67d).
- **News images red**: items without their own image (and channel icons) use `default.jpg`, the logo.
- **Nothing on the lock screen**: qlaunch features items of `priority` 1000 and up with a future `pickup_limit` (a time,
  not a duration). `"featured": true` in a news file does it; only the 3 newest stay featured (the lock screen has 3 slots).
- **Channel icons were a question mark**: qlaunch decodes the icon into a 70x70 texture and rejects any other size;
  icons are the logo scaled to 70x70 (or `<channel>-icon.jpg` / `icon.jpg` in the news folder).
- **"Failed to load" opening a channel**: `online_archives` is a map (`na_required`, `data_list` with a `summary_url`
  per item), not a list.
- **HOME menu title topics empty**: `titles/0100000000001000/topics` answers `nx_news` and `nx_notice` (re-subscribing
  after Clear news is done on the console, by nextendo-nx).

## Credits

- [CrustySean/BCAT-Toolbox](https://github.com/CrustySean/BCAT-Toolbox) (based on Random06457's BCAT-Manager) — News formats.
- The whole Nextendo Network team — https://nextendo.network. Nextendo is awesome.
