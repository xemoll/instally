# 30-system compatibility matrix

Instally now includes a dry-run compatibility matrix:

```bash
instally --compat-matrix
```

This does not modify the host system. It simulates installation planning across 30 common targets by forcing OS, distro family, package manager and architecture profiles.

Covered profiles:

1. CachyOS
2. Arch Linux
3. Manjaro
4. EndeavourOS
5. Garuda
6. Ubuntu
7. Debian
8. Linux Mint
9. Pop!_OS
10. Kali
11. Zorin OS
12. elementary OS
13. Fedora
14. Nobara
15. Rocky Linux
16. AlmaLinux
17. openSUSE Tumbleweed
18. openSUSE Leap
19. Alpine
20. Void Linux
21. Solus
22. Gentoo
23. NixOS
24. Linuxbrew
25. PackageKit fallback
26. Windows winget
27. Windows scoop
28. Windows Chocolatey
29. macOS Homebrew Intel
30. macOS Homebrew Apple Silicon

The matrix checks a mixed list of common programs:

```text
vscode, firefox, discord, telegram, git, curl, node, go, rust, python, java, docker
```

The goal is not to claim that a real VM install was performed. The goal is to catch resolver bugs before installation: wrong manager choice, missing command plan, bad package aliases, private URL mistakes, invalid shell fragments and unsupported source handling.
