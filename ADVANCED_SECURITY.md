# Advanced security checks

Instally security flow:

1. Validate source type and URL.
2. Block private/local download hosts by default.
3. Download to cache using a `.part` file.
4. Calculate SHA-256.
5. Run structure checks: extension/magic mismatch, double extensions, fake AppImage, empty file.
6. Check filesystem permissions and symlinks.
7. Inspect archives for path traversal, symlinks, and archive-bomb indicators.
8. Run embedded signatures including EICAR.
9. Run ClamAV, Defender, or Gatekeeper when available.
10. Run YARA when installed.
11. Run static heuristics for dangerous install scripts.
12. Run VirusTotal hash lookup / URL reputation / optional upload when configured.
13. Install only when checks allow it.

## VirusTotal

Use stdin to avoid saving the key in shell history:

```bash
printf '%s' 'YOUR_KEY' | instally --vt-save-key-stdin
instally --vt-status
instally --vt-test
```

Uploads are off by default:

```bash
instally --scan ./file.AppImage
instally --scan ./file.AppImage --vt-upload
```

## Self-tests

```bash
instally --security-test
scripts/run-quick-mega-checks.sh
```
