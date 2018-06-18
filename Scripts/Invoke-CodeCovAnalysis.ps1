param([string] $CoverFile)

go tool cover -html="$CoverFile" -o "$CoverFile.html"
$Base = ([io.fileinfo]$CoverFile).basename
Write-Host "Interactive HTML report available at: file:///$($pwd)/$Base.html"

$LastLine = (go tool cover -func="$CoverFile")[-1]
$PercentCovered = ($LastLine -split "`t")[-1]

Write-Output $PercentCovered
