param([string] $UnitCoverFile,
      [string] $IntegrationCoverFile)

$CodecovFiles = @($UnitCoverFile, $IntegrationCoverFile)
$MergedCoverFile = "cover_merged.out"

Write-Host "Merging code coverage reports: $CodecovFiles to $MergedCoverFile"

@((Get-Content $CodecovFiles)[0]) + (Get-Content $CodecovFiles | Where-Object { $_ -NotLike "mode:*"}) | Set-Content $MergedCoverFile

