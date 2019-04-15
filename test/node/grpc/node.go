// Copyright 2019 Smart-Edge.com, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/smartedgemec/controller-ce/pb"
	"google.golang.org/grpc"
)

const name = "test-node"

func main() {
	var (
		err  error
		port uint
	)
	log.Print(name, ": starting")

	// CLI flags
	flag.UintVar(&port, "port", 8080, "Port for service to listen on")
	flag.Parse()

	// Set up channels to capture SIGINT and SIGTERM
	sigChan := make(chan os.Signal, 2)
	defer close(sigChan)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create the services
	appSvc := newApplicationService()
	appPolicySvc := newApplicationPolicyService(appSvc)
	appSvc.policyService = appPolicySvc
	vnfSvc := vnfService{}
	interfaceSvc := newInterfaceService()
	ifPolicySvc := newInterfacePolicyService(interfaceSvc)
	interfaceSvc.init(ifPolicySvc)
	zoneSvc := zoneService{}

	// Register the services with the grpc server
	server := grpc.NewServer()
	pb.RegisterApplicationDeploymentServiceServer(server, appSvc)
	pb.RegisterApplicationLifecycleServiceServer(server, appSvc)
	pb.RegisterApplicationPolicyServiceServer(server, appPolicySvc)
	pb.RegisterVNFDeploymentServiceServer(server, &vnfSvc)
	pb.RegisterVNFLifecycleServiceServer(server, &vnfSvc)
	pb.RegisterInterfaceServiceServer(server, interfaceSvc)
	pb.RegisterInterfacePolicyServiceServer(server, ifPolicySvc)
	pb.RegisterZoneServiceServer(server, &zoneSvc)

	// Shut down the server gracefully
	go func() {
		<-sigChan
		log.Printf("%s: shutting down", name)

		server.GracefulStop()
	}()

	// Start the listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal("Error listening on port:", err)
	}
	defer listener.Close()

	// Start the server
	log.Print(name, ": listening on port: ",
		listener.Addr().(*net.TCPAddr).Port)
	err = server.Serve(listener)
	if err != nil && context.Canceled == nil {
		log.Fatal("Error starting GRPC server:", err)
	}
}

func defaultPolicy(appID string) *pb.TrafficPolicy {
	return &pb.TrafficPolicy{
		Id: appID,
		TrafficRules: []*pb.TrafficRule{
			{
				Description: "default_rule",
				Priority:    0,
				Source: &pb.TrafficSelector{
					Description: "default_source",
					Macs: &pb.MACFilter{
						MacAddresses: []string{
							"default_source_mac_0",
							"default_source_mac_1",
						},
					},
				},
				Destination: &pb.TrafficSelector{
					Description: "default_destination",
					Macs: &pb.MACFilter{
						MacAddresses: []string{
							"default_dest_mac_0",
							"default_dest_mac_1",
						},
					},
				},
				Target: &pb.TrafficTarget{
					Description: "default_target",
					Action:      pb.TrafficTarget_ACCEPT,
					Mac: &pb.MACModifier{
						MacAddress: "default_target_mac",
					},
					Ip: &pb.IPModifier{
						Address: "127.0.0.1",
						Port:    9999,
					},
				},
			},
		},
	}
}