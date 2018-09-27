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

# This script is invoked by go generate.
# We bake sample configuration file into the compiled test binary, so that we don't have to
# transfer the file separately to remote machine to invoke integration tests that use it.

$CfgFile = Get-Content -Raw ../contrail-cnm-plugin.conf.sample

"
package configuration_test

const (
    sampleCfgFile = ``
$CfgFile
``
)
" | Out-File baked_data_test.go -Encoding ASCII
# We set encoding to ASCII, because without it, we get 'unexpected NUL in input' error while
# compiling the generated file.
