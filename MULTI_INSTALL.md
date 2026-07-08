# Multi-install

Instally can install many programs in one run. It accepts names, URLs, GitHub repositories, local files, and explicit source prefixes.

```bash
instally --multi "vscode, discord, telegram" --yes
instally --multi vscode --multi discord --multi "github:cli/cli" --yes
```

A text list also works:

```text
vscode
discord
telegram
github: cli/cli
https://example.com/app.AppImage
```

Run it:

```bash
instally --batch apps.txt --dry-run --yes
instally --batch apps.txt --yes
```

During safe install, URL/GitHub assets are downloaded into cache, scanned, and then installed from the checked cache file. System packages use package-manager trust because there is no standalone file before installation.

## Continue after errors

For a long install list, use:

```bash
instally --multi "vscode, discord, telegram, github:cli/cli" --yes --continue-on-error
```

Without `--continue-on-error`, Instally stops after the first failed command. With it, Instally keeps processing the rest of the list and records failures in the final output.
