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
param([switch] $DryRun)

function Get-HeaderFromTemplate($CommentCharacters, $LicenseTemplate) {
    $Today = Get-Date
    $Updated = $LicenseTemplate -replace "<year>",$Today.Year
    $Commented = ""
    foreach($Line in ($Updated -split [Environment]::NewLine)) {
        if ($Line -ne "") {
            $Commented += "$CommentCharacters $Line$([Environment]::NewLine)"
        } else {
            $Commented += "$CommentCharacters$([Environment]::NewLine)"
        }
    }
    return $Commented
}

function Find-FilesThatShouldBeLicensed($FileExtension) {
    Get-ChildItem -Path . -Recurse -Filter "*$FileExtension" | Where-Object {$_.FullName -NotMatch "vendor"}
}

function Select-FilesWithInvalidLicenseHeader($AllFiles) {
    $InvalidFiles = @()
    foreach($File in $AllFiles) {
        Write-Host (Resolve-Path -Relative $File.FullName) -NoNewLine
        $Content = Get-Content -Raw $File.FullName
        $Match = $Content | Select-String "All Rights Reserved" -AllMatches
        if (-not $Match) {
            Write-Host "... invalid license header"
            $InvalidFiles += $File
        } else {
            Write-Host "... OK"
        }
    }
    return $InvalidFiles
}

function Add-LicenseHeaderToFiles($Files, $LicenseHeader, $DryRun) {
    foreach($File in $Files) {
        $Content = Get-Content -Raw $File.FullName
        $NewContent = $LicenseHeader + $Content
        if ($DryRun) {
            Write-Host "DRY RUN: $(Resolve-Path -Relative $File.FullName)" -NoNewLine
            Write-Host $NewContent
        } else {
            Set-Content -Path $File.FullName -Value $NewContent `
                -NoNewLine # don't add newline at end of file
        }
    }
}

function Add-LicenseToFilesOfLanguage($FileExtension, $CommentCharacters, $Template, $DryRun) {
    $LicenseHeader = Get-HeaderFromTemplate $CommentCharacters $Template
    $LanguageFiles = Find-FilesThatShouldBeLicensed $FileExtension
    $InvalidLanguageFiles = Select-FilesWithInvalidLicenseHeader $LanguageFiles
    Add-LicenseHeaderToFiles $InvalidLanguageFiles $LicenseHeader $DryRun
}

$LicenseHeaderTemplate = '
 Copyright (c) <year> Juniper Networks, Inc. All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the `"License`");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an `"AS IS`" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
'

Add-LicenseToFilesOfLanguage ".go" "//" $LicenseHeaderTemplate $DryRun
Add-LicenseToFilesOfLanguage ".ps1" "#" $LicenseHeaderTemplate $DryRun
