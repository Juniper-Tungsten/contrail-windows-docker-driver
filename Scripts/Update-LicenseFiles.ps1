#
# Looks for license files in vendor/ directory and generates two files:
# - Attributions.txt
# - LICENSE_MSI.txt
#
# The script should be invoked whenever updating dependencies using `dep ensure` or similar.
#
param([string] $AttributionsPath="Attributions.txt",
      [string] $LicenseMSIPath="LICENSE_MSI.txt",
      [string] $OurLicensePath="LICENSE")

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function New-EmptyFile($Path) {
    if (Test-Path $Path) {
        Remove-Item $Path | Out-Null
    }
    New-Item -Type File -Path $Path | Out-Null
}

function Write-File($Msg, $File) {
    Write-Output $Msg | Out-File -FilePath $File -Append -Encoding Ascii
}

function Find-VendorLicenseFiles {
    Get-ChildItem -Path ".\vendor\*" -Recurse -Filter "LICENSE*"
}

function Group-PackagesByLicenseHash($LicenseFiles) {
    $HashToFiles = @{}
    foreach($File in $LicenseFiles) {
        $HashObj = Get-FileHash -Path $File -Algorithm MD5
        $Hash = $HashObj.Hash
        $Relpath = $HashObj.Path | Resolve-Path -Relative
        Write-Host "$Relpath"
        if ($HashToFiles.Keys -contains $Hash) {
            $HashToFiles[$Hash] += $Relpath
        } else {
            $HashToFiles[$Hash] = @($Relpath)
        }
    }
    return $HashToFiles
}

function Write-AttributionsFile($HashToFiles, $File) {
    foreach($Hash in $HashToFiles.Keys) {
        Write-File -File $File @"
Start files:
 $($HashToFiles[$Hash] -join "`r`n ")
End files

Start Copyright text:

$(Get-Content -Raw $HashToFiles[$Hash][0])

End Copyright text
======================================================================
"@
    }
}

function Write-LicenseMSIFile($LicenseFiles, $OurLicense, $File) {
    $AllLicenses = @($OurLicense) + $LicenseFiles
    foreach($License in $AllLicenses) {
        Write-File -File $File @"
$(Get-Content -Raw $License)

    ======================================================================

"@
    }
}

$Licenses = Find-VendorLicenseFiles
$GroupedLicenses = Group-PackagesByLicenseHash $Licenses

New-EmptyFile($AttributionsPath)
Write-AttributionsFile $GroupedLicenses $AttributionsPath

New-EmptyFile($LicenseMSIPath)
Write-LicenseMSIFile $Licenses $OurLicensePath $LicenseMSIPath
