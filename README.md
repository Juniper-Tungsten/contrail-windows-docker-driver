# contrail-windows-docker-driver

## Prerequisites

### Third party dependencies

1. Download `dep` package manager for golang: https://github.com/golang/dep

2. Fetch dependencies into vendor/ directory:
```
dep ensure -v
dep prune -v
```

### Generate Contrail Go API

1. Checkout github.com/Juniper/contrail-api-client to some temporary directory

```
git clone https://github.com/Juniper/contrail-api-client C:\some_dir\
```

2. Generate Golang API

```
C:\some_dir\contrail-api-client\generateds\generateDS.py -q -f -o C:\some_dir\types\ -g golang-api C:\some_dir\contrail-api-client\schema\vnc_cfg.xsd
```

3. Copy generated files to specific third party vendor package

```
cp C:\some_dir\types\* .\vendor\github.com\Juniper\contrail-go-api\types\
```

## Building

```
.\Invoke-Build.ps1 [-SkipExe] [-SkipTests] [-SkipMSI]
```

Build artifacts will appear under build/ directory.

## Testing

Running the following script will execute tests and collect code coverage statistics.

```
.\Invoke-Tests.ps1 [-RunIntegrationTests]
```
