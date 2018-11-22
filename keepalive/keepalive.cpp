#include <iostream>
#include <memory>
#include <string>

#include <grpcpp/grpcpp.h>
#include "keepalive.grpc.pb.h"

using grpc::Server;
using grpc::ServerBuilder;
using grpc::ServerContext;
using grpc::Status;
using keepalive::Hello;
using keepalive::Response;
using keepalive::KeepAlive;

class KeepAliveServiceImpl final : public KeepAlive::Service{
    Status alive (ServerContext* context, const Hello* hello, Response* reply) override {
        std::cout << "Received a remote priority of " << hello->priority() << std::endl;
        reply->set_status(true);
        reply->set_priority(1);
        return Status::OK;
    }
};


void RunServer() {
    std::string server_address("0.0.0.0:5000");
    KeepAliveServiceImpl service;

    ServerBuilder builder;

    // Listen
    builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);
    std::unique_ptr<Server> server(builder.BuildAndStart());
    std::cout << "Server listening on " << server_address << std::endl;

    server->Wait();
}

int main(int argc, char** argv) {
    RunServer();
    return 0;
}