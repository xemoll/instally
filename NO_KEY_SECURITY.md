# No-key security mode

VirusTotal is useful, but the official API requires an API key. Instally therefore treats VirusTotal as optional:

- no key: local/system checks continue;
- with key: VirusTotal hash lookup and optional upload are added;
- uploads are off by default.

No-key checks currently include:

- SHA-256 calculation;
- embedded EICAR test signature detection;
- ClamAV via `clamdscan` or `clamscan` when installed;
- Microsoft Defender command-line scan on Windows;
- macOS Gatekeeper / `spctl` checks;
- optional YARA checks when `yara` is installed;
- static script heuristics;
- package-manager trust mode for repository installs.

Run a safe detection self-test:

```bash
instally --security-test
```

Run multi-install without a VirusTotal key:

```bash
instally --multi "vscode, discord, telegram, github:cli/cli" --dry-run --yes
instally --multi "vscode, discord, telegram, github:cli/cli" --yes
```

For a stronger local setup on Linux, install ClamAV and refresh signatures before relying on local scans.

## Expanded offline checks

When VirusTotal is not configured, Instally now still performs several local checks:

- SHA-256 hash calculation;
- EICAR embedded signature detection;
- ClamAV/Microsoft Defender/Gatekeeper when available;
- YARA when installed;
- file magic and extension consistency checks;
- empty installer and world-writable file warnings;
- zip/tar path traversal checks;
- archive symlink/link and archive-bomb heuristics;
- script/static heuristics for suspicious install patterns.

These checks are not a guarantee that a file is safe. They are designed to block obvious dangerous cases and make unknown/risky cases visible before installation.
