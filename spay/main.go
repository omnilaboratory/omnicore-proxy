package main

import (
	"context"
	"crypto/tls"
	"flag"
	"github.com/lightningnetwork/lnd/cert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net"
	"om-rpc-tool/signal"
	"om-rpc-tool/toolrpc"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var certDir = ""
	var LuckServerPort = "38332"
	var lnddir = "false"
	var nodeAddress = ""
	var dbConnstr = ""
	var netType = ""
	flag.StringVar(&netType, "net_type", "", "")
	flag.StringVar(&certDir, "cert_dir", "./cert", "tls cert dir")
	flag.StringVar(&LuckServerPort, "luck_server_port", "38332", "luck package service port")
	flag.StringVar(&lnddir, "lnddir", "", "lnddir contain macaroon-asset for connect oblndNode ")
	flag.StringVar(&nodeAddress, "node_address", "localhost:10001", "oblndNode grpc address")
	flag.StringVar(&dbConnstr, "db_conn", "localhost:10001", "mysql connstr : user:password@tcp(127.0.0.1:3306)/db?charset=utf8&parseTime=True&loc=Local")
	flag.Parse()

	shutdownChan, err := signal.Intercept()
	if dbConnstr != "" {
		toolrpc.InitDb(dbConnstr)
	} else {
		panic("miss dbConnstr")
	}
	if netType == "" {
		log.Fatalln("miss net_type")
	}
	if len(lnddir) == 0 {
		log.Fatalln("miss lnddir")
	}
	if certDir != "" {
		os.MkdirAll(certDir, os.ModePerm)
	}

	//grpcs
	lis1, err := net.Listen("tcp", ":"+LuckServerPort)
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}
	tlsCfg := getTlsCfg(certDir)
	tlsCfg.ClientAuth = tls.RequireAnyClientCert
	serverCreds := credentials.NewTLS(tlsCfg)
	serverOpts := []grpc.ServerOption{grpc.Creds(serverCreds),
		grpc.UnaryInterceptor(useridInterceptor),
		grpc.StreamInterceptor(useridInterceptorStream),
	}
	luckServer := toolrpc.NewLuckPkServer(nodeAddress, netType, lnddir, &shutdownChan)
	s1 := grpc.NewServer(serverOpts...)
	toolrpc.RegisterLuckPkApiServer(s1, luckServer)
	log.Println("Serving gRPCs on 0.0.0.0:" + LuckServerPort)
	go func() {
		log.Fatalln(s1.Serve(lis1))
	}()
	<-shutdownChan.ShutdownChannel()
	luckServer.Wg.Wait()
	log.Println("app exit")
	return
}

func getTlsCfg(certDir string) *tls.Config {
	certPath := filepath.Join(certDir, "tls.cert")
	keyPath := filepath.Join(certDir, "tls.key")
	if !fileExists(certPath) {
		log.Println("Generating TLS certificates...")
		err := cert.GenCertPair(
			"lnd autogenerated cert", certPath,
			keyPath, nil, nil, true,
			1000*24*time.Hour,
		)
		if err != nil {
			log.Fatalln(err)
		}
		log.Println("Done generating TLS certificates")
	}
	certData, _, err := cert.LoadCert(
		certPath, keyPath,
	)
	if err != nil {
		log.Println("err load TLS certificates")
	}
	//tlsCfg := cert.TLSConfFromCert(certData)
	return &tls.Config{Certificates: []tls.Certificate{certData}}
	//return tlsCfg
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func useridInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	//log.Println("useridInterceptor ", info.FullMethod)
	if info.FullMethod == "/toolrpc.luckPkApi/RegistTlsKey" {
		return handler(ctx, req)
	}

	userid, err := toolrpc.GetUserIdKey(ctx)
	if err != nil {
		return nil, err
	}
	newCtx := toolrpc.SetGrpcHeader(ctx, "userid", userid)
	return handler(newCtx, req)
}

func useridInterceptorStream(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := stream.Context()
	userid, err := toolrpc.GetUserIdKey(ctx)
	if err != nil {
		return err
	}
	newCtx := toolrpc.SetGrpcHeader(ctx, "userid", userid)
	wrapped := toolrpc.WrapServerStream(stream)
	wrapped.WrappedContext = newCtx
	return handler(srv, wrapped)
}
