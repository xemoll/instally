
## Audit hardening update

- Dark native UI toned down: less neon, shorter labels, safer layout for long text.
- URL downloads now block local/private hosts by default to reduce SSRF/local-network risk. Use INSTALLY_ALLOW_PRIVATE_URLS=1 only when you intentionally install from a trusted LAN/local source.
- GitHub source-build fallback is blocked unless --allow-unknown is passed.
- Release asset scoring now respects forced/target architecture via INSTALLY_FORCE_ARCH.
- Local safe dry-run now shows the real installation plan after scanning.

# Instally Native GUI

The main desktop interface is now the native Go/Fyne app under `native/fyne`.

The UI is intentionally minimal:

- one source input field;
- one quiet file picker action;
- one primary action: `Проверить и установить`;
- compact progress steps: `Источник → Загрузка → Проверка → Установка`;
- advanced options and logs are collapsed by default.

The start screen does not contain promotional text. Human-readable explanations appear during checking, installation, and result rendering.

## Run

```bash
cd native/fyne
go mod tidy
go run .
```

## Build

```bash
./build-native.sh
```

The native GUI uses the same Go installer core as the CLI. It does not use HTML, CSS, WebView, or a localhost server.

## Overflow-safe dark UI

The native Fyne interface now uses a constrained vertical layout. Long URLs, checksums and command lines are compacted in the visible cards and remain available in the Journal. This avoids text crossing into buttons, step labels or result cards.

Main visible flow:

1. Paste URL / GitHub / app name or choose file.
2. Click **Проверить и установить**.
3. Instally scans available file/source data.
4. If safe, it installs from the already checked cache/source.
5. Full details stay in **Журнал**.

For diagnostics:

```bash
instally --doctor
instally --support
```
