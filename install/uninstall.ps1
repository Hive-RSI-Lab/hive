# Uninstall the Hive (your folder, ~\hive by default, is left alone):
#   irm https://raw.githubusercontent.com/ihsan-sa/hive-standalone-build/main/install/uninstall.ps1 | iex
$dir = Join-Path $env:LOCALAPPDATA 'Programs\hive'
$exe = Join-Path $dir 'hive.exe'
if (Test-Path $exe) { & $exe uninstall }
Start-Sleep -Seconds 1
Remove-Item $dir -Recurse -Force -ErrorAction SilentlyContinue
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
[Environment]::SetEnvironmentVariable('Path', (($userPath -split ';') | Where-Object { $_ -ne $dir }) -join ';', 'User')
Write-Host 'The Hive is uninstalled. Your folder and its projects are untouched.'
