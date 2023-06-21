package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/btcsuite/btcd/rpcclient"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"om-rpc-tool/toolrpc"
	"os/exec"
	"strings"
)

type RpcServer struct {
	toolrpc.UnimplementedToolsServer
	btcClient *rpcclient.Client
	//regtest testnet mainnet
	NetType string
}

func NewRpc(btcClient *rpcclient.Client, netType string) *RpcServer {
	return &RpcServer{btcClient: btcClient, NetType: netType}
}

func (s *RpcServer) GetBalance(ctx context.Context, req *toolrpc.OmniGetbalanceReq) (*toolrpc.OmniGetbalanceRes, error) {
	//return &toolrpc.OmniGetbalanceRes{}, nil
	return OmniGetbalance(req, s.btcClient)
}

func (s *RpcServer) ChannelSend(ctx context.Context, req *toolrpc.OmniSendCoinReq) (*toolrpc.OmniSendCoinRes, error) {
	if s.NetType == "mainnet" {
		return nil, nil
	}
	cmdstr := "scripts/channel_send.sh %s %d"
	req.Address = strings.Replace(req.Address, ";", "", -1)
	cmdstr = fmt.Sprintf(cmdstr, req.Address, req.AssetId)
	args := strings.Fields(cmdstr)
	cmd := exec.Command(args[0], args[1:]...)
	log.Println("cmd.string", cmd.String())
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		err = errors.New(err.Error() + out.String())
		return nil, err
	}
	return &toolrpc.OmniSendCoinRes{Result: out.String()}, nil
}
func (s *RpcServer) SendCoin(ctx context.Context, req *toolrpc.OmniSendCoinReq) (*toolrpc.OmniSendCoinRes, error) {
	cmdstr := "scripts/send_coin.sh %s %d"
	if s.NetType == "regtest" { //proxy model
		//this is none docker version ; send_coin.sh invoke the omnicore-cli to sendcoin and mine block
	} else {
		//this is docker version ; docker_send_coin.sh invoke the docker's omnicore-cli to sendcoin
		cmdstr = "scripts/docker/send_coin.sh %s %d"
	}
	req.Address = strings.Replace(req.Address, ";", "", -1)
	cmdstr = fmt.Sprintf(cmdstr, req.Address, req.AssetId)
	log.Println(cmdstr)
	args := strings.Fields(cmdstr)
	cmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		err = errors.New(err.Error() + out.String())
		return nil, err
	}

	return &toolrpc.OmniSendCoinRes{Result: out.String()}, nil
}
func execCmd(cmdstr string) (string, error) {
	log.Println(cmdstr)
	args := strings.Fields(cmdstr)
	cmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		err = errors.New(err.Error() + out.String())
		return "", err
	}
	return out.String(), err
}
func (s *RpcServer) Mine(context.Context, *emptypb.Empty) (*toolrpc.OmniMineCoinRes, error) {
	res, err := execCmd("mine.sh")
	if err != nil {
		return nil, err
	}
	return &toolrpc.OmniMineCoinRes{Result: res}, err
}

func (s *RpcServer) GetProperty(ctx context.Context, req *toolrpc.OmniGetPropertyReq) (*toolrpc.OmniGetPropertyRes, error) {
	return OmniGetProperty(req, s.btcClient)
}
func (s *RpcServer) ListProperties(ctx context.Context, req *emptypb.Empty) (*toolrpc.ListPropertiesRes, error) {
	return OmniListproperties(req, s.btcClient)
}
func (s *RpcServer) CreateProperty(ctx context.Context, req *toolrpc.CreatePropertyReq) (*toolrpc.CreatePropertyRes, error) {
	return OmniCreateProperty(req, s.btcClient)
}
