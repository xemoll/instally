# Universal install mode

Instally now treats a request as a source-resolution problem:

1. known app alias or preset;
2. system package manager when the app is a normal package;
3. Flatpak/Snap for cross-distro GUI apps;
4. winget/scoop/choco on Windows;
5. Homebrew/MacPorts on macOS;
6. GitHub Release for projects with release assets;
7. direct URL or local installer file with download -> scan -> install;
8. source clone/build only when explicitly allowed.

## Multi-install

```bash
instally --multi "vscode, discord, telegram, firefox, brave, obs, vlc, blender, gimp, krita, steam, docker, node, go, rust" --dry-run --yes
instally --batch apps.txt --yes --continue-on-error
```

## Presets

```bash
instally --preset base --dry-run --yes
instally --preset dev --dry-run --yes
instally --preset gaming --dry-run --yes
instally --preset media --dry-run --yes
instally --preset work --dry-run --yes
```

Presets are only shortcuts. The same safety rules still apply.

## Safety defaults

- direct URLs are downloaded to cache first;
- local/private URLs are blocked unless `INSTALLY_ALLOW_PRIVATE_URLS=1`;
- local installers are scanned before install;
- unknown GitHub source-build fallback is blocked unless `--allow-unknown`;
- Flatpak automatically ensures the Flathub remote before installing;
- multi-install can continue after errors with `--continue-on-error`.
