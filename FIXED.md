# Fixed — nextendo-bcat-nx

- **News "Find channels" TLS refused**: its own certificate covering `*.cdn.nintendo.net` (BCAT checks the name).
- **Containers rejected**: header version byte set to 1, as bcat requires.
- **Requests never logged**: every request logged with its status.
- **News**: catalog, channels, lists and items in the console's formats, encrypted; items are JSON files (NEWS.md).
- **News lists ignored (no item ever downloaded)**: `service_status` is `in_service`; bcat 22.5.0 rejects anything but
  `in_service`/`expired` (0x67d).
- **News images red**: items without their own image (and channel icons) use `default.jpg`, the logo.
- **No news after Clear news**: the HOME menu title (`titles/0100000000001000/topics`) answers `nx_news` and `nx_notice`,
  so a console re-subscribes to the default feed.

## Credits

- [CrustySean/BCAT-Toolbox](https://github.com/CrustySean/BCAT-Toolbox) (based on Random06457's BCAT-Manager) — News formats.
- The whole Nextendo Network team — https://nextendo.network. Nextendo is awesome.
