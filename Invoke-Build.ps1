#
#  Copyright (c) 2018 Juniper Networks, Inc. All Rights Reserved.
#
#  Licensed under the Apache License, Version 2.0 (the `"License`");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an `"AS IS`" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
#
param([switch] $SkipExe,
      [switch] $SkipTests,
      [switch] $SkipMSI)

$OutDir = "$pwd\build"

Write-Host "Starting build. Output dir = $OutDir"

if (-not $SkipExe) {
    Write-Host "** Building .exe..."
    go build -o "$OutDir\contrail-cnm-plugin.exe" -v .

    if ($LastExitCode -ne 0) {
        throw "Build failed."
    }
    Copy-Item "$OutDir\contrail-cnm-plugin.exe" "$OutDir\contrail-windows-docker.exe"
} else {
    Write-Host "** Skipping building of .exe"
}

if (-not $SkipTests) {
    Write-Host "** Building tests..."
    ./Scripts/New-BakedTestData.ps1

    ginkgo build -r -tags "unit integration" .
    if ($LastExitCode -ne 0) {
        throw "Build failed."
    }

    Get-ChildItem -Recurse -Filter "*.test" | `
        ForEach-Object { Move-Item -Force -Path $_.FullName -Destination (Join-Path $OutDir ($_.Name + ".exe")) }
} else {
    Write-Host "** Skipping building of tests"
}

if (-not $SkipMSI) {
    Write-Host "** Building .MSI..."
    candle.exe -nologo -I -o "$OutDir\msi.wixobj" msi.wxs
    light.exe -nologo -out "$OutDir\contrail-cnm-plugin.msi" "$OutDir\msi.wixobj"

    if ($LastExitCode -ne 0) {
        throw "Build failed."
    }
    Copy-Item "$OutDir\contrail-cnm-plugin.msi" "$OutDir\docker-driver.msi"
} else {
    Write-Host "** Skipping building of MSI"
}
