param([switch] $SkipExe,
      [switch] $SkipTests,
      [switch] $SkipMSI)

$OutDir = "$pwd/build"

Write-Host "Starting build. Output dir = $OutDir"

if (-not $SkipExe) {
    Write-Host "** Building .exe..."
    go build -o (Join-Path $OutDir "contrail-windows-docker.exe") -v .

    if ($LastExitCode -ne 0) {
        throw "Build failed."
    }
} else {
    Write-Host "** Skipping building of .exe"
}

if (-not $SkipTests) {
    Write-Host "** Building tests..."
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
    go-msi make --arch x64 --version 0.1 --keep `
        --msi (Join-Path $OutDir "docker-driver.msi") `
        --path "./wix.json" `
        --src "./template" `
        --license "./LICENSE_MSI.txt" `
        --out (Join-Path $OutDir "gomsi")

    if ($LastExitCode -ne 0) {
        throw "Build failed."
    }
} else {
    Write-Host "** Skipping building of MSI"
}
