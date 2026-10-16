<#
.SYNOPSIS
    Pretends to build a project. Does nothing real: it prints what a build would print and waits.

.DESCRIPTION
    A stand-in for trying out the project_hub script runner. It restores, compiles, tests and packages
    an imaginary project, writes progress as it goes, and exits with 0 or, with -Fail, with 1.
#>
[CmdletBinding()]
param(
    [ValidateSet('Debug', 'Release')]
    [string]$Configuration = 'Release',

    [string]$Version = '0.0.0-dev',

    # Milliseconds to wait between steps, to make the output arrive gradually.
    [int]$DelayMs = 400,

    # Make the test step fail, to see what a failed run looks like.
    [switch]$Fail
)

$ErrorActionPreference = 'Stop'
$modules = 'core', 'api', 'ui', 'operator', 'charts'
$started = Get-Date

function Step([string]$Message) {
    Write-Output ''
    Write-Output "==> $Message"
    Start-Sleep -Milliseconds $DelayMs
}

Write-Output "Simulated build of example-project $Version ($Configuration)"
Write-Output "Working directory: $(Get-Location)"
if ($env:BUILD_NUMBER) { Write-Output "BUILD_NUMBER from environment: $env:BUILD_NUMBER" }
if ($env:BUILD_TARGET) { Write-Output "BUILD_TARGET from this script's own variables: $env:BUILD_TARGET" }
# set by each user on the Settings tab; the config lists it under environment.recommended as required
if ($env:DEPLOY_USER) { Write-Output "Deploying as: $env:DEPLOY_USER" }
else { Write-Warning 'DEPLOY_USER is not set; set it on the Settings tab (simulated check)' }
foreach ($name in 'BUILD_BANNER', 'RELEASE_NOTES') {
    # values with several lines arrive with their line breaks
    $value = [Environment]::GetEnvironmentVariable($name)
    if ($value) {
        Write-Output "${name}:"
        $value -split "`n" | ForEach-Object { Write-Output "    $_" }
    }
}

Step 'Restoring dependencies'
foreach ($module in $modules) {
    Write-Output "    restored $module"
}

Step 'Compiling'
$index = 0
foreach ($module in $modules) {
    $index++
    Write-Output ("    [{0}/{1}] {2} -> out/{3}/{2}.dll" -f $index, $modules.Count, $module, $Configuration)
    Start-Sleep -Milliseconds $DelayMs
}
Write-Warning 'api: 2 obsolete calls (simulated warning)'

Step 'Running tests'
$passed = 128
if ($Fail) {
    Write-Output "    passed: $($passed - 3), failed: 3"
    Write-Error 'Tests failed (simulated). Build stopped.' -ErrorAction Continue
    exit 1
}
Write-Output "    passed: $passed, failed: 0"

Step 'Packaging'
Write-Output "    out/example-project-$Version.tgz"

$seconds = [math]::Round(((Get-Date) - $started).TotalSeconds, 1)
Write-Output ''
Write-Output "Build succeeded in $seconds s (nothing was actually built)."
exit 0
