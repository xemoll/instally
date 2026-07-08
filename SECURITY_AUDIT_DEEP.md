# Instally deep security audit hardening

This release hardens installation flows instead of only adding more app aliases.

## Fixed vulnerability classes

1. Plain HTTP downloads are blocked by default. Use HTTPS, or explicitly set `INSTALLY_ALLOW_INSECURE_HTTP=1` only for a trusted local mirror.
2. Insecure Git schemes are blocked by default. `https://` and `ssh://` are allowed; `http://` and `git://` require `INSTALLY_ALLOW_INSECURE_GIT=1`.
3. Archive path traversal, symlinks/hardlinks inside archives, and archive-bomb indicators are now hard `unsafe` findings, not soft warnings.
4. Official script installers no longer use broad `--allow-unknown`. They use a dedicated `--trusted-official-script` path that accepts only exact allowlisted HTTPS URLs.
5. VirusTotal keys are no longer forwarded to child Instally processes through environment variables. Saved 0600 config remains the recommended path.
6. Download URL DNS/private-host validation now uses a short resolver timeout to avoid installer hangs on broken DNS.
7. File/cache names are sanitized more strictly so generated paths cannot become empty, traversal-like, or odd hidden names.
8. GitHub release commands no longer duplicate `--vt-upload`.
9. Dry-run URL installs never download the file; they show the expected cache path, scan step, and install plan.
10. Warnings-only plans still fail instead of pretending that a rejected install succeeded.

## Trusted official script policy

`--trusted-official-script` is for internal use by AI-tools installers. It accepts only:

- `https://ollama.com/install.sh`
- `https://claude.ai/install.sh`

The file is still downloaded into cache and scanned. The trusted mode only allows a limited non-unsafe scan result for these exact URLs.

## Compatibility check

The compatibility matrix remains dry-run only. It checks plan generation for 30 common OS/package-manager profiles without mutating the current system.
