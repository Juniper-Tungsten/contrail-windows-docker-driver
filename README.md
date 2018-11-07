# contrail-windows-docker-driver

## Prerequisites

### Third party dependencies

#. Download `dep` package manager for golang: https://github.com/golang/dep

#. Fetch dependencies into vendor/ directory:
    ```
    dep ensure -v
    ```

### Generate Contrail Go API

1. Checkout github.com/Juniper/contrail-api-client to some temporary directory

    ```
    git clone https://github.com/Juniper/contrail-api-client C:\some_dir\
    ```

#. Generate Golang API

    Requirements: python version <= 2.7.13, lxml package

    ```
    virtualenv -p \path\to\python<=2.7.13\executable path\to\env
    \path\to\env\Scripts\activate
    pip install lxml
    C:\some_dir\contrail-api-client\generateds\generateDS.py -q -f -o C:\some_dir\types\ -g golang-api C:\some_dir\contrail-api-client\schema\vnc_cfg.xsd
    ```

#. Copy generated files to specific third party vendor package

    ```
    cp C:\some_dir\types\* .\vendor\github.com\Juniper\contrail-go-api\types\
    ```

## Building

1. Install go packages

    ```
    go get github.com/onsi/ginkgo/ginkgo
    go get github.com/onsi/gomega/...
    ```

#. Install WiX toolset

    See: https://github.com/wixtoolset/wix3/releases/tag/wix3111rtm

    Make sure to add WiX to your PATH variable (for example: `C:\Program Files (x86)\WiX Toolset v3.11\bin`)
    Note that WiX's MSBuild requires .NET 3.5 to be installed. See section 'WiX system requirements':
    http://wixtoolset.org/documentation/manual/v3/main/

#. Install chocolatey

    See: https://chocolatey.org/install

#. Invoke build

    ```
    .\Invoke-Build.ps1 [-SkipExe] [-SkipTests] [-SkipMSI]
    ```

Build artifacts will appear under build/ directory.

## Testing

Running the following script will execute tests and collect code coverage statistics.

```
.\Invoke-Tests.ps1 [-RunIntegrationTests]
```
