# Fixed — nextendo-bcat-nx

- **News "Find channels" TLS refused**: its own certificate covering `*.cdn.nintendo.net` (BCAT checks the name).
- **Containers rejected**: header version byte set to 1, as bcat requires.
- **Requests never logged**: every request logged with its status.
- **News**: catalog, channels, lists and items in the console's formats, encrypted; items are JSON files (NEWS.md).

## Credits

- [CrustySean/BCAT-Toolbox](https://github.com/CrustySean/BCAT-Toolbox) (based on Random06457's BCAT-Manager) — News formats.
- The whole Nextendo Network team — https://nextendo.network. Nextendo is awesome.
