# GitHub release checklist

Use this checklist when publishing a new Instally build.

## Validation

```bash
go test ./... -count=1 -timeout=180s
go vet ./...
gofmt -l cmd internal
cd native/fyne && go test ./... && go vet ./...
cd ../..
./instally --security-test
./instally --compat-matrix
./instally --update firefox discord lazygit --dry-run
./instally --upgrade-all --dry-run
```

## Release assets

Expected downloadable files:

- `instally-linux-amd64.tar.gz`
- `instally-windows-amd64.zip`
- `instally-darwin-amd64.tar.gz`
- `instally-darwin-arm64.tar.gz`
- `SHA256SUMS.txt`

## Safety expectations

- plain names resolve through trusted sources only;
- no random search-engine result downloads;
- URL and local installers are checked before use;
- unsafe results are blocked;
- source builds require explicit opt-in.

## Recommended GitHub layout

For a public release-only repository:

- keep README, SECURITY, release notes and downloadable archives public;
- keep full development source in a separate private repository;
- publish stable builds as GitHub Release assets when the release upload API or GitHub CLI is available.
