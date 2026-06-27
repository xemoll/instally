# Security policy

Instally is designed to avoid the common "download the first search result" installer problem.

## Supported security model

- Plain names are resolved only through known app profiles, official allowlists, GitHub Release API selection, or trusted system package managers.
- Instally does not use Google, Yandex, Bing, ad results, SEO pages, mirrors or random download portals for automatic installation.
- Downloaded/local/GitHub files are scanned before install and copied to a private verified cache with SHA-256 recheck.
- Plain HTTP and private/local URL downloads are blocked by default.
- Source builds are blocked by default.

## Reporting issues

Please open a private security advisory or issue with:

- Instally version/commit
- operating system and package manager
- exact command/input
- security report output, with API keys/secrets removed

Never paste VirusTotal API keys or other credentials into public issues.
