# Checks Set-UserPath in install.ps1 and uninstall.ps1 against the real user PATH,
# then puts it back: a PATH it has no reason to change stays as stored, and a change
# keeps %VAR% entries and the REG_EXPAND_SZ type. Windows only: pwsh install/path_check.ps1
$ErrorActionPreference = 'Stop'
$k = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment')
$saved = $k.GetValue('Path', $null, 'DoNotExpandEnvironmentNames')
$savedKind = if ($null -ne $saved) { $k.GetValueKind('Path') }
# Each case starts from its own PATH; the trailing ';' shows an untouched value.
function Case([string]$Remove, [string]$Add, [string]$Want, [string]$Why) {
  $k.SetValue('Path', '%USERPROFILE%\bin;C:\Tools;', 'ExpandString')
  Set-UserPath -Remove $Remove -Add $Add
  $got = $k.GetValue('Path', $null, 'DoNotExpandEnvironmentNames')
  if ($got -cne $Want -or $k.GetValueKind('Path') -ne 'ExpandString') { throw "${script} ${Why}: PATH is '$got' ($($k.GetValueKind('Path')))" }
}
try {
  foreach ($script in 'install.ps1', 'uninstall.ps1') {
    $ast = [Management.Automation.Language.Parser]::ParseFile((Join-Path $PSScriptRoot $script), [ref]$null, [ref]$null)
    $fn = $ast.Find({ param($a) $a -is [Management.Automation.Language.FunctionDefinitionAst] -and $a.Name -eq 'Set-UserPath' }, $true)
    if (-not $fn) { throw "$script has no Set-UserPath" }
    Invoke-Expression $fn.Extent.Text
    Case -Remove 'C:\Not\There' -Want '%USERPROFILE%\bin;C:\Tools;' -Why 'rewrote a PATH without the folder'
    Case -Add 'C:\Tools' -Want '%USERPROFILE%\bin;C:\Tools;' -Why 'rewrote a PATH that has the folder'
    Case -Add 'C:\Hive' -Want '%USERPROFILE%\bin;C:\Tools;C:\Hive' -Why 'added the folder badly'
    Case -Remove 'C:\Tools' -Want '%USERPROFILE%\bin' -Why 'removed the folder badly'
    Write-Host "$script keeps the user PATH as stored"
  }
} finally {
  if ($null -ne $saved) { $k.SetValue('Path', $saved, $savedKind) } else { $k.DeleteValue('Path', $false) }
  $k.Close()
}
