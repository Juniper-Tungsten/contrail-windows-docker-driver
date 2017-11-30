#
# Copyright (c) 2017 Juniper Networks, Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# # For manual testing you can uncomment this and run a 
# # python -m SimpleHTTPServer 9090 in another terminal
# # the tests should pass.
# import socket
# print "Hello"
# s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
# s.connect(("127.0.0.1", 9090))
# s.send("Test msg pls ignore")
# s.close()

import sys
from contrail_vrouter_api.vrouter_api import ContrailVRouterApi

PORT_TYPE = 'NovaVMPort'


def main():
    operation = sys.argv[1]

    # TODO: ContrailVRouterApi has semaphores, but we call it like a batch command.
    # Therefore, we need to make sure that proper synchronization is realized in driver, so that
    # add/delete operations don't overlap.
    api = ContrailVRouterApi()

    if operation == "add":
        try:
            (operation, vm_uuid, vif_uuid, if_name, mac, docker_id, ip_address, vn_uuid) = sys.argv[1:]
            if_name_noquotes = if_name.replace('"', '')
            api.add_port(vm_uuid, vif_uuid, if_name_noquotes, mac, port_type=PORT_TYPE,
                         display_name=docker_id, ip_address=ip_address, vn_id=vn_uuid)
        except Exception:
            print("{}: 'add' exception caught: re raise")
            raise
    elif operation == "delete":
        try:
            (operation, vifUuid) = sys.argv[1:]
            api.delete_port(vifUuid)
        except Exception:
            print("MOCK not reporting FAIL, since agent is not implemented yet.")
    else:
        raise ValueError("Invalid operation. Must be add or delete")


if __name__ == "__main__":
    main()