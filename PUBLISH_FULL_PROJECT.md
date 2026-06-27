# Full project publication

This repository is prepared for full Instally publication.

The current ChatGPT GitHub connector can write repository text files, but it cannot create GitHub Releases or upload binary release assets. For a complete release, use the prepared publish package generated during the release process.

## What the full publisher does

The publisher script:

1. clones `xemoll/instally`;
2. replaces the repository contents with the full source tree;
3. adds CI and release workflow files;
4. runs the local validation suite;
5. commits and pushes the full project;
6. creates or updates GitHub Release `v1.0.0`;
7. uploads Linux, Windows and macOS archives;
8. verifies SHA-256 checksums.

## Validation suite

```bash
go test ./... -count=1 -timeout=180s
go vet ./...
go build -trimpath -o /tmp/instally-publish-check ./cmd/instally
/tmp/instally-publish-check --security-test
/tmp/instally-publish-check --compat-matrix
/tmp/instally-publish-check --update firefox discord lazygit --dry-run
/tmp/instally-publish-check --upgrade-all --dry-run
cd native/fyne && go test ./... && go vet ./...
```

## Expected final user flow

Users should be able to:

```bash
git clone https://github.com/xemoll/instally.git
cd instally
go test ./...
go build -o instally ./cmd/instally
./instally --doctor
./instally --security-test
./instally firefox --dry-run
```

And non-developers should be able to download ready archives from:

```text
https://github.com/xemoll/instally/releases/tag/v1.0.0
```

## Release assets

Expected assets:

- `instally-linux-amd64.tar.gz`
- `instally-windows-amd64.zip`
- `instally-darwin-amd64.tar.gz`
- `instally-darwin-arm64.tar.gz`
- `SHA256SUMS.txt`
