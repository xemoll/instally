# VirusTotal integration

Instally uses VirusTotal as an additional reputation layer. It never replaces a local antivirus, package signatures, or manual review.

## Configure

Temporary for one run:

```bash
INSTALLY_VT_API_KEY=YOUR_KEY instally --scan ./app.AppImage
instally --vt-key YOUR_KEY --scan ./app.AppImage
```

Save for the current user:

```bash
instally --vt-save-key YOUR_KEY
instally --vt-status
instally --vt-clear-key
```

The saved key is stored in the user config with `0600` permissions. Instally masks secrets in command output.

## Upload policy

By default Instally performs hash lookup only. Unknown files are not uploaded unless you explicitly enable it:

```bash
instally --scan ./app.AppImage --vt-upload
INSTALLY_VT_UPLOAD=1 instally --install-local-safe ./app.AppImage --yes
```

Upload limits:

- up to 32 MB: direct `/api/v3/files` upload;
- above 32 MB: Instally requests `/api/v3/files/upload_url` and streams the file there;
- default maximum: 650 MB;
- override lower maximum: `INSTALLY_VT_MAX_UPLOAD_MB=128`.

Do not upload private documents, secrets, tokens, private builds, or anything you do not want shared with security vendors.

## URL flow

For URL installs Instally can check URL reputation first, then download to cache, scan the downloaded file, and only then install:

```bash
instally --install-url-safe https://example.com/app.AppImage --yes
```

Private and localhost URLs are blocked by default to reduce SSRF/local-network risks. Override only when you know what you are doing:

```bash
INSTALLY_ALLOW_PRIVATE_URLS=1 instally --install-url-safe http://127.0.0.1/app.AppImage --yes
```

## Safer key saving

Avoid putting your API key directly into shell history. Prefer:

```bash
printf '%s' 'PASTE_NEW_KEY_HERE' | instally --vt-save-key-stdin
```

The key is stored in the user config file with `0600` permissions. Instally masks key/token environment variables in command previews and logs.

## Key test

```bash
instally --vt-test
```

This checks the configured key through a safe EICAR hash lookup path. It does not upload a file.
