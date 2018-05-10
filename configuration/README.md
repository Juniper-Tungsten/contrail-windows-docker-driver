These tests require, that sample config file in the root of this repository is baked
into the binary itself. Supplied `Invoke-Build.ps1` and `Invoke-Tests.ps1` scripts already
generate the required code. To do this by hand, however, invoke
```
go generate -tags unit,integration ./...
```
