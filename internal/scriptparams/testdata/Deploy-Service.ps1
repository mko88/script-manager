<#
.SYNOPSIS
    Deploys a service to one or more environments.

.DESCRIPTION
    Builds the image and rolls it out.

.PARAMETER Service
    Name of the service to deploy.

.PARAMETER Environments
    Where to deploy it.

.PARAMETER Tags
    Extra image tags.
.PARAMETER DryRun
    Print what would happen.
#>
#requires -Version 7
[CmdletBinding(SupportsShouldProcess)]
param(
    [Parameter(Mandatory, Position = 0, HelpMessage = 'ignored, help wins')]
    [string]$Service,

    [Parameter(Mandatory = $false)]
    [ValidateSet('dev', 'staging', "prod")]
    [string[]]$Environments = @('dev', 'staging'),

    [string[]]$Tags,

    [ValidateSet('fast', 'safe')]
    [string]$Strategy = 'safe', # trailing comment

    [Parameter(HelpMessage = "Replica count")]
    [int]$Replicas = (Get-Default),

    [switch]$DryRun
)

function Inner { param([string]$NotMe) }
Write-Output $Service
