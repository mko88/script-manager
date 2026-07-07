# Builds script-manager from the Windows host by delegating to the Go dev
# container: there's no Go toolchain on the host, so `bash build.sh` has to
# run inside the devcontainer for this repo. Wraps two steps that otherwise
# have to be done by hand every time:
#   1. Stop any running script-manager*.exe — a locked binary makes the
#      Windows cross-compile step in build.sh fail with "permission denied"
#      when it tries to overwrite bin/*.exe.
#   2. Find that devcontainer (its name is auto-generated and changes across
#      recreations, so it's matched by the "devcontainer.local_folder" label
#      instead of a hardcoded name) and run build.sh inside it.
#
# Builds both platforms by default, matching build.sh — pass -Windows or
# -Linux to build only that platform (-Windows for routine use on this
# host, which never runs the Linux binaries; -Linux when only a Linux
# binary is needed, e.g. Xvfb-based visual verification inside the
# container).
param(
    [switch]$Windows,
    [switch]$Linux
)

$ErrorActionPreference = "Stop"

$repoRoot = $PSScriptRoot.TrimEnd('\')

Get-Process script-manager*, sm-config-edit* -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

$container = $null
foreach ($name in (docker ps --format "{{.Names}}")) {
    $folder = docker inspect $name --format '{{ index .Config.Labels "devcontainer.local_folder" }}' 2>$null
    if ($folder -and ($folder.TrimEnd('\') -ieq $repoRoot)) {
        $container = $name
        break
    }
}
if (-not $container) {
    Write-Error "No running devcontainer found for $repoRoot. Open this repo's devcontainer in VS Code first."
    exit 1
}
Write-Host "Using container: $container"

$buildArgs = if ($Windows) { "--windows" } elseif ($Linux) { "--linux" } else { "" }
docker exec $container bash -c "cd /workspaces/script-manager && bash build.sh $buildArgs"
exit $LASTEXITCODE
