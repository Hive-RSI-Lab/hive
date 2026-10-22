# Install the Hive on Windows:
#   irm https://raw.githubusercontent.com/ihsan-sa/hive-standalone-build/main/install/install.ps1 | iex
# Running it again reinstalls. Agents are sandboxed inside WSL 2, so the Linux
# build is installed next to the Windows one.
$ErrorActionPreference = 'Stop'
$repo = if ($env:HIVE_REPO) { $env:HIVE_REPO } else { 'ihsan-sa/hive-standalone-build' }
$dir = Join-Path $env:LOCALAPPDATA 'Programs\hive'
$base = "https://github.com/$repo/releases/latest/download"
$tmp = Join-Path $env:TEMP ("hive-install-" + [guid]::NewGuid())
New-Item -ItemType Directory -Force $dir, $tmp | Out-Null

Write-Host 'Downloading the Hive...'
Invoke-WebRequest "$base/checksums.txt" -OutFile "$tmp\checksums.txt" -UseBasicParsing
foreach ($a in @('hive-windows-amd64.exe', 'hive-linux-amd64')) {
  Invoke-WebRequest "$base/$a" -OutFile "$tmp\$a" -UseBasicParsing
  $line = Select-String -Path "$tmp\checksums.txt" -Pattern " $([regex]::Escape($a))$" | Select-Object -First 1
  if (-not $line) { throw "no checksum for $a" }
  $want = $line.Line.Split(' ')[0]
  $got = (Get-FileHash "$tmp\$a" -Algorithm SHA256).Hash.ToLower()
  if ($want -ne $got) { throw "checksum mismatch for $a; not installing" }
}

$exe = Join-Path $dir 'hive.exe'
if (Test-Path $exe) { Move-Item $exe "$exe.old" -Force }   # a running hive.exe can be renamed, not overwritten
Copy-Item "$tmp\hive-windows-amd64.exe" $exe -Force
Copy-Item "$tmp\hive-linux-amd64" (Join-Path $dir 'hive-linux-amd64') -Force
Remove-Item $tmp -Recurse -Force

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (($userPath -split ';') -notcontains $dir) {
  [Environment]::SetEnvironmentVariable('Path', "$userPath;$dir", 'User')
  Write-Host "Added $dir to your PATH (open a new terminal)."
}
& $exe --version
Write-Host 'Installed. Run: hive'
