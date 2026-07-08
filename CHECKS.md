
## Audit hardening update

- Dark native UI toned down: less neon, shorter labels, safer layout for long text.
- URL downloads now block local/private hosts by default to reduce SSRF/local-network risk. Use INSTALLY_ALLOW_PRIVATE_URLS=1 only when you intentionally install from a trusted LAN/local source.
- GitHub source-build fallback is blocked unless --allow-unknown is passed.
- Release asset scoring now respects forced/target architecture via INSTALLY_FORCE_ARCH.
- Local safe dry-run now shows the real installation plan after scanning.

# Checks

The project was checked after the UI implementation pass:

```bash
go test ./...
go build -o instally ./cmd/instally
GOOS=windows GOARCH=amd64 go build -o dist/windows-amd64/instally.exe ./cmd/instally
GOOS=darwin GOARCH=amd64 go build -o dist/darwin-amd64/instally ./cmd/instally
GOOS=darwin GOARCH=arm64 go build -o dist/darwin-arm64/instally ./cmd/instally
GOOS=linux GOARCH=amd64 go build -o dist/linux-amd64/instally ./cmd/instally
bash -n install.sh install-full.sh uninstall.sh install-macos.sh
./instally --text 'github: https://github.com/cli/cli/releases/latest' --dry-run --yes
./instally --text 'https://example.com/app.AppImage' --dry-run --yes
```

Real package installation and VirusTotal uploads were not executed in the build container.
