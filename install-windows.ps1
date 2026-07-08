$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
$Bin = Join-Path $Root "instally.exe"
if (!(Test-Path $Bin)) { $Bin = Join-Path $Root "dist\windows-amd64\instally.exe" }
if (!(Test-Path $Bin)) { throw "Build first: go build -o instally.exe ./cmd/instally" }
& $Bin --install-self --yes
