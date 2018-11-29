#
# Copyright (c) 2018 Juniper Networks, Inc. All rights reserved.
#

$ErrorActionPreference = "Stop"

function Initialize-Logger {
    New-Item -ItemType Directory -Path C:\ProgramData\Contrail\var\log\contrail -Force
}

function Write-Log {
    Param (
        [Parameter(Mandatory=$True)] [string] $Message
    )

    $Logfile = "C:\ProgramData\Contrail\var\log\contrail\contrail-autostart.log"
    Add-Content $Logfile -Value $Message
}

function Wait-CnmPluginPipe {
    function Test-CnmPluginPipe {
        $PipePath = "//./pipe/Contrail"
        return (Test-Path $PipePath)
    }

    $MaxAttempts = 5
    $TimeoutInSeconds = 20

    $TimesToGo = $MaxAttempts
    While ((-not (Test-CnmPluginPipe)) -and ($TimesToGo -gt 0)) {
        Start-Sleep -Seconds $TimeoutInSeconds
        $TimesToGo = $TimesToGo - 1
    }

    if (Test-CnmPluginPipe) {
        Write-Log "Waiting for CNM plugin pipe succeeded"
        return $True
    } else {
        Write-Log "Waiting for CNM plugin pipe failed"
        return $False
    }
}

function Remove-AllContainers {
    function Get-Containers {
        return (docker ps -aq)
    }

    Write-Log "Removing all Docker containers"

    $MaxAttempts = 3

    # Sometimes we have to retry removing containers due to a bug in Docker/HCS
    $TimesToGo = $MaxAttempts
    $Containers = Get-Containers
    While ($Containers -and ($TimesToGo -gt 0)) {
        docker rm -f @Containers
        $Containers = Get-Containers
        $TimesToGo = $TimesToGo - 1
    }
    if ($Containers) {
        Write-Log "Removing containers failed"
    } else {
        Write-Log "Removing containers succeeded"
    }
}

function Remove-AgentPorts {
    Write-Log "Removing Agent port files"

    $PortsPath = "C:\ProgramData\Contrail\var\lib\contrail\ports"
    Remove-Item -Path $PortsPath -Recurse -Force -ErrorAction SilentlyContinue
}

function Remove-HnsNetworks {
    Write-Log "Removing HNS Networks"

    Get-ContainerNetwork | Remove-ContainerNetwork -Force -ErrorAction SilentlyContinue
    Get-ContainerNetwork | Remove-ContainerNetwork -Force
}

function Initialize-ComputeNode {
    Initialize-Logger
    Write-Log "Entering contrail-autostart"

    Stop-Service docker
    Remove-HnsNetworks
    Remove-AgentPorts

    Start-Service docker
    Remove-AllContainers

    Write-Log "Starting contrail-cnm-plugin"
    Start-Service contrail-cnm-plugin

    if (Wait-CnmPluginPipe) {
        Write-Log "Starting contrail-vrouter-agent"
        Start-Service contrail-vrouter-agent
        return $True
    } else {
        return $False
    }
}

Try {
    if (Initialize-ComputeNode) {
        Write-Log "contrail-autostart succeeded"
    } else {
        Write-Log "contrail-autostart failed"
    }
} Catch {
    $ErrorMessage = $_.Exception.Message
    Write-Log "contrail-autostart failed with the following error message: $ErrorMessage"
}
