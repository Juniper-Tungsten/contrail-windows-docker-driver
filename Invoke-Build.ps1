param([string] $SrcPath = ".",
      [string] $Out = "contrail-windows-docker.exe",
      [string] $OutDir = ".",
      [switch] $BuildTests)


$OutDirAbs = Join-Path $pwd $OutDir
$ExeOutPath = Join-Path $OutDirAbs $Out

go build -v -o $ExeOutPath $SrcPath

if ($BuildTests) {
    try {
        Push-Location $SrcPath

        ginkgo build -r

        Get-ChildItem -Recurse -Filter "*.test" | `
            ForEach-Object { Move-Item -Path $_.FullName -Destination ($OutDirAbs + "/" + $_.Name + ".exe") }
    } finally {
        Pop-Location
    }
}
