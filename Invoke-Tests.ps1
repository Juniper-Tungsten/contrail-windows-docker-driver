param([switch] $RunIntegrationTests)

Write-Host "Running docker driver unit tests..."
go test -coverpkg=./... -covermode count -coverprofile="cover_unit.out" -tags unit .

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
