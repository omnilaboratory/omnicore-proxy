package main

import (
	"context"
	"flag"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"log"
	"net"
	"om-rpc-tool/gateway"
	"om-rpc-tool/toolrpc"
	"om-rpc-tool/toolrpc/server"
)

func main() {
	var btcHost = ""
	flag.Set("logtostderr", "true")
	flag.StringVar(&btcHost, "btc_host", "43.138.107.248:8332", "ip:port")
	flag.Parse()

	// Connect to local namecoin core RPC server using HTTP POST mode.
	connCfg := &rpcclient.ConnConfig{
		Host:         btcHost,
		User:         "omniwallet",
		Pass:         "cB3]iL2@eZ1?cB2?",
		HTTPPostMode: true, // Namecoin core only supports HTTP POST mode
		DisableTLS:   true, // Namecoin core does not provide TLS by default
	}
	// Notice the notification parameter is nil since notifications are
	// not supported in HTTP POST mode.
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Shutdown()

	rserver := server.NewRpc(client)

	// Create a listener on TCP port
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// Create a gRPC server object
	s := grpc.NewServer()
	// Attach the Greeter service to the server
	toolrpc.RegisterToolsServer(s, rserver)
	// Serve gRPC Server
	log.Println("Serving gRPC on 0.0.0.0:8080")
	go func() {
		log.Fatalln(s.Serve(lis))
	}()

	//gw server
	opts := gateway.Options{
		Addr: ":8090",
		GRPCServer: gateway.Endpoint{
			Network: "tcp",
			Addr:    "0.0.0.0:8080",
		},
		OpenAPIDir: "swagger",
	}
	gateway.Run(context.Background(), opts, []func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error{toolrpc.RegisterToolsHandler})

	//simpest
	// Create a client connection to the gRPC server we just started
	// This is where the gRPC-Gateway proxies the requests
	//conn, err := grpc.DialContext(
	//	context.Background(),
	//	"0.0.0.0:8080",
	//	grpc.WithBlock(),
	//	grpc.WithTransportCredentials(insecure.NewCredentials()),
	//)
	//if err != nil {
	//	log.Fatalln("Failed to dial server:", err)
	//}
	//
	//gwmux := runtime.NewServeMux()
	//// Register Greeter
	//err = toolrpc.RegisterToolsHandler(context.Background(), gwmux, conn)
	//if err != nil {
	//	log.Fatalln("Failed to register gateway:", err)
	//}
	//
	//gwServer := &http.Server{
	//	Addr:    ":8090",
	//	Handler: gwmux,
	//}
	//
	//log.Println("Serving gRPC-Gateway on http://0.0.0.0:8090")
	//log.Fatalln(gwServer.ListenAndServe())
}
