#!/usr/bin/env python3

import glass_pb2 as pb
import glass_pb2_grpc
import grpc
import sys


# Set up GRPC server details
grpcserver = "127.0.0.1:7181"
channel = grpc.insecure_channel(grpcserver)
stub = glass_pb2_grpc.looking_glassStub(channel)

# Get info
address = pb.ip_address(
    address = sys.argv[1],
)
a = pb.origin_request(
    ip_address = address,
)

try:
    response = stub.origin(a)
except grpc.RpcError as e:
    #print("Error: {}".format(e))
    print("Error: {}".format(e.details()))
    sys.exit(1)

print("The origin of {} is {}".format(sys.argv[1], response.origin_asn))

try:
    resp = stub.route(a)
except grpc.RpcError as e:
    #print("Error: {}".format(e))
    print("Error: {}".format(e.details()))
    sys.exit(1)

#print(resp)
print("The active route for {} is {}/{}".format(sys.argv[1], resp.ip_address.address, resp.ip_address.mask))

try:
    resp = stub.aspath(a)
except grpc.RpcError as e:
    #print("Error: {}".format(e))
    print("Error: {}".format(e.details()))
    sys.exit(1)

#print(resp)
for i in range(len(resp.asn)):
    print("asn #{}: {}".format(i+1, resp.asn[i]))


asn = pb.asname_request(
    as_number = response.origin_asn,
)

try:
    resp = stub.asname(asn)
except grpc.RpcError as e:
    #print("Error: {}".format(e))
    print("Error: {}".format(e.details()))
    sys.exit(1)

print("The ASNAME for AS{} is {}".format(response.origin_asn, resp.as_name))