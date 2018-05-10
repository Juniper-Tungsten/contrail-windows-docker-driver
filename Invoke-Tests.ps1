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
param([switch] $RunIntegrationTests)

./Scripts/New-BakedTestData.ps1

Write-Host "Running docker driver unit tests..."
go test -coverpkg=./... -covermode count -coverprofile="cover_unit.out" -tags unit . -- ginkgo.trace

if ($RunIntegrationTests) {
    Write-Host "Running docker driver integration tests..."
    go test -coverpkg=./... -covermode count -coverprofile="cover_integration.out" -tags integration .

    .\Scripts\Merge-CoverFiles.ps1 -UnitCoverFile "cover_unit.out" `
        -IntegrationCoverFile "cover_integration.out"

    $Unit = .\Scripts\Invoke-CodeCovAnalysis.ps1 -CoverFile "cover_unit.out"
    $Int = .\Scripts\Invoke-CodeCovAnalysis.ps1 -CoverFile "cover_integration.out"
    $Total = .\Scripts\Invoke-CodeCovAnalysis.ps1 -CoverFile "cover_merged.out"
    
} else {
    $Unit = .\Scripts\Invoke-CodeCovAnalysis.ps1 -CoverFile "cover_unit.out"
    $Int = "N/A"
    $Total = "N/A"
}

$Summary = @"
========================= CODE COVERAGE SUMMARY =========================
Coverage of modules with unit tests: $Unit
Coverage of modules with integration tests: $Int
Total coverage (unit + integration tests): $Total
"@

$Summary | Set-Content "codecov_summary.txt"
Write-Host $Summary
