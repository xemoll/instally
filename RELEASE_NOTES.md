# Instally v1.0.0

## Highlights

Instally v1.0.0 focuses on safer app resolution, verified downloads and predictable dry-run/update flows.

Main changes:

- trusted resolver for plain application names;
- no automatic Google, Yandex, Bing or random download portal fallback;
- verified cache for selected local files, URL downloads and GitHub release assets;
- new update preview commands:
  - `instally --update firefox discord lazygit --dry-run`
  - `instally --upgrade-all --dry-run`
- protected local web interface with token-based API access;
- dependency-light native launcher included in builds.

## Verification

The build was checked with:

```text
go test ./... -count=1 -timeout=180s
go vet ./...
native/fyne go test / go vet
go build ./cmd/instally
instally --security-test
instally --compat-matrix
instally --update firefox discord lazygit --dry-run
instally --upgrade-all --dry-run
```

All listed checks passed in the prepared release build.
