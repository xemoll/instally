package app

import "strings"

func normalizeNativePackagesForManager(manager string, items []string) []string {
	out := make([]string, 0, len(items)+4)
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item))
		mapped := nativePackageAliases(manager, key)
		if len(mapped) == 0 {
			mapped = []string{item}
		}
		out = append(out, mapped...)
	}
	return appendUnique(nil, out...)
}

func nativePackageAliases(manager, key string) []string {
	aliases := map[string]map[string][]string{
		"pacman": {
			"python": {"python"}, "python-pip": {"python-pip"}, "jdk-openjdk": {"jdk-openjdk"}, "java": {"jdk-openjdk"},
			"node": {"nodejs", "npm"}, "nodejs": {"nodejs", "npm"}, "rust": {"rust"}, "go": {"go"},
		},
		"apt": {
			"python": {"python3"}, "python-pip": {"python3-pip"}, "jdk-openjdk": {"default-jdk"}, "java": {"default-jdk"},
			"node": {"nodejs", "npm"}, "nodejs": {"nodejs", "npm"}, "rust": {"rustc", "cargo"}, "go": {"golang-go"},
			"docker": {"docker.io"}, "virtualbox": {"virtualbox"}, "wine": {"wine"},
		},
		"dnf": {
			"python": {"python3"}, "python-pip": {"python3-pip"}, "jdk-openjdk": {"java-21-openjdk-devel"}, "java": {"java-21-openjdk-devel"},
			"node": {"nodejs", "npm"}, "nodejs": {"nodejs", "npm"}, "rust": {"rust", "cargo"}, "go": {"golang"},
			"docker": {"moby-engine"},
		},
		"zypper": {
			"python": {"python3"}, "python-pip": {"python3-pip"}, "jdk-openjdk": {"java-21-openjdk-devel"}, "java": {"java-21-openjdk-devel"},
			"node": {"nodejs", "npm"}, "nodejs": {"nodejs", "npm"}, "rust": {"rust", "cargo"}, "go": {"go"}, "docker": {"docker"},
		},
		"apk": {
			"python": {"python3"}, "python-pip": {"py3-pip"}, "jdk-openjdk": {"openjdk21"}, "java": {"openjdk21"},
			"node": {"nodejs", "npm"}, "nodejs": {"nodejs", "npm"}, "rust": {"rust", "cargo"}, "go": {"go"}, "docker": {"docker"},
		},
		"xbps": {
			"python": {"python3"}, "python-pip": {"python3-pip"}, "jdk-openjdk": {"openjdk21"}, "java": {"openjdk21"},
			"node": {"nodejs", "npm"}, "nodejs": {"nodejs", "npm"}, "rust": {"rust", "cargo"}, "go": {"go"}, "docker": {"docker"},
		},
		"eopkg": {
			"python": {"python3"}, "python-pip": {"python3-pip"}, "jdk-openjdk": {"openjdk-21"}, "java": {"openjdk-21"},
			"node": {"nodejs", "npm"}, "nodejs": {"nodejs", "npm"}, "rust": {"rust", "cargo"}, "go": {"golang"}, "docker": {"docker"},
		},
		"emerge": {
			"python": {"dev-lang/python"}, "jdk-openjdk": {"virtual/jdk"}, "java": {"virtual/jdk"},
			"node": {"net-libs/nodejs"}, "nodejs": {"net-libs/nodejs"}, "rust": {"dev-lang/rust"}, "go": {"dev-lang/go"}, "docker": {"app-containers/docker"},
		},
		"nix": {
			"python": {"python3"}, "python-pip": {"python3Packages.pip"}, "jdk-openjdk": {"jdk21"}, "java": {"jdk21"},
			"node": {"nodejs"}, "nodejs": {"nodejs"}, "rust": {"rustc", "cargo"}, "go": {"go"},
		},
		"brew": {
			"python": {"python"}, "python-pip": {"python"}, "jdk-openjdk": {"openjdk"}, "java": {"openjdk"},
			"node": {"node"}, "nodejs": {"node"}, "rust": {"rust"}, "go": {"go"},
		},
	}
	if byManager, ok := aliases[manager]; ok {
		return byManager[key]
	}
	return nil
}
