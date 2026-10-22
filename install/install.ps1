# Install the Hive on Windows. One line:
#   gh api repos/ihsan-sa/hive-standalone-build/contents/install/install.ps1 -H "Accept: application/vnd.github.raw" | Out-String | iex
# (or, once the repo is public: irm https://raw.githubusercontent.com/ihsan-sa/hive-standalone-build/main/install/install.ps1 | iex)
# Running it again reinstalls. Agents are sandboxed inside WSL 2, so the Linux
# build is installed next to the Windows one, and Claude Code is installed or
# updated inside WSL.
$ErrorActionPreference = 'Stop'
$repo = if ($env:HIVE_REPO) { $env:HIVE_REPO } else { 'ihsan-sa/hive-standalone-build' }
$cmd = if ($env:HIVE_CMD) { $env:HIVE_CMD } else { 'hive-v0' }
$dir = if ($env:HIVE_BIN_DIR) { $env:HIVE_BIN_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\$cmd" }
$tmp = Join-Path $env:TEMP ('hive-install-' + [guid]::NewGuid())
New-Item -ItemType Directory -Force $dir, $tmp | Out-Null
$assets = @('checksums.txt', 'hive-windows-amd64.exe', 'hive-linux-amd64')

Write-Host "Downloading the Hive from $repo ..."
$useGh = $false
if (Get-Command gh -ErrorAction SilentlyContinue) {
  & gh auth status *> $null
  $useGh = ($LASTEXITCODE -eq 0)
}
foreach ($a in $assets) {
  if ($useGh) {
    & gh release download -R $repo --pattern $a --dir $tmp --clobber
    if ($LASTEXITCODE -ne 0) { throw "could not download $a (is there a release?)" }
  } else {
    Invoke-WebRequest "https://github.com/$repo/releases/latest/download/$a" -OutFile (Join-Path $tmp $a) -UseBasicParsing
  }
}
foreach ($a in $assets[1..2]) {
  $line = Select-String -Path (Join-Path $tmp 'checksums.txt') -Pattern " $([regex]::Escape($a))$" | Select-Object -First 1
  if (-not $line) { throw "no checksum for $a" }
  $want = $line.Line.Split(' ')[0]
  $got = (Get-FileHash (Join-Path $tmp $a) -Algorithm SHA256).Hash.ToLower()
  if ($want -ne $got) { throw "checksum mismatch for $a; not installing" }
}

$exe = Join-Path $dir "$cmd.exe"
if (Test-Path $exe) {
  & $exe shutdown *> $null   # the next start runs the new version; agents keep running
  Move-Item $exe "$exe.old" -Force   # a running exe can be renamed, not overwritten
}
Copy-Item (Join-Path $tmp 'hive-windows-amd64.exe') $exe -Force
Copy-Item (Join-Path $tmp 'hive-linux-amd64') (Join-Path $dir 'hive-linux-amd64') -Force
Remove-Item $tmp -Recurse -Force

if (-not $env:HIVE_NO_PATH) {
  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  if (($userPath -split ';') -notcontains $dir) {
    [Environment]::SetEnvironmentVariable('Path', "$userPath;$dir", 'User')
  }
  if (($env:Path -split ';') -notcontains $dir) { $env:Path = "$env:Path;$dir" }
}

# Agents run inside WSL 2 with Claude Code installed there.
if (-not $env:HIVE_SKIP_WSL) {
  if (Get-Command wsl.exe -ErrorAction SilentlyContinue) {
    Write-Host 'Installing or updating Claude Code inside WSL (agents run there, sandboxed)...'
    & wsl.exe --exec bash -lc 'if [ -x $HOME/.local/bin/claude ]; then $HOME/.local/bin/claude update; else curl -fsSL https://claude.ai/install.sh | bash; fi'
    if ($LASTEXITCODE -ne 0) { Write-Warning "Claude Code inside WSL didn't install; `"$cmd doctor`" will say what to do." }
  } else {
    Write-Warning "WSL 2 isn't installed. Agents need it for their sandbox: run 'wsl --install -d Ubuntu' (then restart), or opt into unsandboxed runs with '$cmd doctor --bare'."
  }
}

& $exe --version
Write-Host ''
Write-Host "Installed. Run: $cmd"
