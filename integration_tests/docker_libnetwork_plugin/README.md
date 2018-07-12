This package tests integration of docker daemon and docker libnetwork plugin.

This includes:
* interacting with actual docker daemon to create networks and containers
* (actually needs to pull container images)
* starting cnm API server on a named pipe for docker daemon to talk to

To run the tests, user should provide the following parameters:
* `netAdapter`
* `vswitchNameWildcard`

See `help` for more information.

CAUTION: tests are known to be a little bit flaky due to HNS (windows container networking).
