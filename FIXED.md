# Fixed — nextendo-bcat-nx

- **News "Find channels" TLS refused**: its own certificate covering `*.cdn.nintendo.net` (BCAT checks the name).
- **Containers rejected**: header version byte set to 1, as bcat requires.
- **Requests never logged**: every request logged with its status.
- **News**: catalog, channels, lists and items in the console's formats, encrypted; items are JSON files (NEWS.md).
- **News lists ignored (no item ever downloaded)**: `service_status` is `in_service`; bcat 22.5.0 rejects anything but
  `in_service`/`expired` (0x67d).
- **News images red**: items without their own image (and channel icons) use `default.jpg`, the logo.
- **Nothing on the lock screen**: qlaunch features only `priority` > 50 with a future `pickup_limit` (a time, not a
  duration); items default to priority 1500 and are featured for 14 days after their date (shown on the lock screen).
- **HOME menu title topics empty**: `titles/0100000000001000/topics` answers `nx_news` and `nx_notice` (re-subscribing
  after Clear news is done on the console, by nextendo-nx).

## Credits

- [CrustySean/BCAT-Toolbox](https://github.com/CrustySean/BCAT-Toolbox) (based on Random06457's BCAT-Manager) — News formats.
- The whole Nextendo Network team — https://nextendo.network. Nextendo is awesome.
