# ff-installer

A tiny Windows program that puts a shortcut for a Frenchfries777 web app (game, lesson or tool) in the
Start menu and on the Desktop, and opens it in its own window (Microsoft Edge or Google Chrome `--app` mode).

This repository contains only the installer. The apps themselves (games, lessons, tools, logos) are **not** part
of this project and are not covered by its license; they are available at https://frenchfries777.com.

## Download

Get `setup.exe` from the [Releases page](https://github.com/frenchfries777mc/ff-installer/releases), rename it to
`<app-id>-setup.exe` (for example `stickseum-setup.exe`) and run it. This project uses free code signing provided by [SignPath.io](https://signpath.io), certificate by SignPath Foundation
(signing is being set up; v1.0.0 is not signed yet). See "Code signing policy" below.
The normal way to get it is the install page of each app on https://frenchfries777.com.

## What it does

The file is named `<app-id>-setup.exe` (for example `stickseum-setup.exe`). When you run it:

1. It asks for confirmation, telling you which shortcuts it will create.
2. It downloads the app's public web manifest `https://frenchfries777.com/games/pwa/<app-id>.webmanifest`
   and the app's icon from the same site (nothing else is downloaded, no other program is run except your browser).
3. It creates two shortcuts, one in the Start menu and one on the Desktop, and stores the app icon in
   `%LOCALAPPDATA%\Frenchfries777\<app-id>\icon.ico`.
4. It opens the app.

## Uninstall

Delete the two shortcuts (Start menu and Desktop) and the folder `%LOCALAPPDATA%\Frenchfries777\<app-id>`,
or run `<app-id>-setup.exe /uninstall`.

## Build

```
sh scripts/build.sh
```

Requires Go 1.24 or newer. The binary is built by GitHub Actions (`.github/workflows/build.yml`).

## Privacy policy

This program does not collect, store or send personal data. Its only network requests are two HTTPS downloads
from frenchfries777.com (the app manifest and icon) described above. The web site's own privacy practices
are described on that site.

## Code signing policy

Free code signing provided by [SignPath.io](https://signpath.io), certificate by [SignPath Foundation](https://signpath.org).

- Committers and reviewers: Frenchfries777 (repository owner)
- Approvers: Frenchfries777
- Only binaries built by the GitHub Actions workflow in this repository from this source are signed.

## License

MIT, see `LICENSE`.
