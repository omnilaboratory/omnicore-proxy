package server

import (
	"encoding/json"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/rpcclient"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"om-rpc-tool/toolrpc"
)

// OmniGetbalanceCmd defines the OmniGetbalance JSON-RPC command.
type OmniGetbalanceCmd struct {
	Address    string
	PropertyId int
}

// OmniGetbalanceResult models the data from the Omni-Getbalance command.
type OmniGetbalanceResult struct {
	Balance  string `json:"balance"`
	Reserved string `json:"reserved"`
	Frozen   string `json:"frozen"`
}

// futureOmniGetbalanceRes is a future promise to deliver the result of a
// OmniGetbalanceAsync RPC invocation (or an applicable error).
type futureOmniGetbalanceRes chan *rpcclient.Response

// Receive waits for the response promised by the future and returns a
// transaction given its hash.
func (r futureOmniGetbalanceRes) Receive() (*toolrpc.OmniGetbalanceRes, error) {
	res, err := rpcclient.ReceiveFuture(r)
	if err != nil {
		return nil, err
	}

	// take care of the special case where the output has been spent already
	// it should return the string "null"
	if string(res) == "null" {
		return nil, nil
	}
	log.Println(string(res), err)
	// Unmarshal result as an OmniGetbalance result object.
	var txOutInfo *toolrpc.OmniGetbalanceRes
	err = json.Unmarshal(res, &txOutInfo)
	if err != nil {
		return nil, err
	}

	return txOutInfo, nil
}

// OmniGetbalanceAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
func OmniGetbalanceAsync(req *toolrpc.OmniGetbalanceReq, c *rpcclient.Client) futureOmniGetbalanceRes {
	cmd := &OmniGetbalanceCmd{Address: req.Address, PropertyId: int(req.PropertyId)}
	log.Println(req)
	return c.SendCmd(cmd)
}

// OmniGetbalance Returns the token balance for a given address and property.
func OmniGetbalance(req *toolrpc.OmniGetbalanceReq, c *rpcclient.Client) (*toolrpc.OmniGetbalanceRes, error) {
	return OmniGetbalanceAsync(req, c).Receive()
}

func init() {
	err := btcjson.RegisterCmd("omni_getbalance", (*OmniGetbalanceCmd)(nil), btcjson.UFWalletOnly)
	if err != nil {
		panic(err)
	}
}

// OmniGetPropertyCmd defines the omniGetProperty JSON-RPC command.
type OmniGetPropertyCmd struct {
	PropertyId int
}

// OmniGetPropertyRes models the data from the Omni-Getbalance command.
//type OmniGetPropertyRes struct {
//	Propertyid      int    `json:"propertyid"`
//	Name            string `json:"name"`
//	Category        string `json:"category"`
//	Subcategory     string `json:"subcategory"`
//	Data            string `json:"data"`
//	Url             string `json:"url"`
//	Divisible       bool   `json:"divisible"`
//	Issuer          string `json:"issuer"`
//	Creationtxid    string `json:"creationtxid"`
//	Fixedissuance   bool   `json:"fixedissuance"`
//	Managedissuance bool   `json:"managedissuance"`
//	Totaltokens     string `json:"totaltokens"`
//}

// futureomniGetPropertyRes is a future promise to deliver the result of a
// omniGetPropertyAsync RPC invocation (or an applicable error).
type futureomniGetPropertyRes chan *rpcclient.Response

// Receive waits for the response promised by the future and returns a
// transaction given its hash.
func (r futureomniGetPropertyRes) Receive() (*toolrpc.OmniGetPropertyRes, error) {
	res, err := rpcclient.ReceiveFuture(r)
	if err != nil {
		return nil, err
	}

	// take care of the special case where the output has been spent already
	// it should return the string "null"
	if string(res) == "null" {
		return nil, nil
	}
	// Unmarshal result as an omniGetProperty result object.
	var txOutInfo *toolrpc.OmniGetPropertyRes
	err = json.Unmarshal(res, &txOutInfo)
	if err != nil {
		return nil, err
	}

	return txOutInfo, nil
}

// omniGetPropertyAsync returns an instance of a type that can be used to get
// the result of the RPC at some future time by invoking the Receive function on
// the returned instance.
func omniGetPropertyAsync(req *toolrpc.OmniGetPropertyReq, c *rpcclient.Client) futureomniGetPropertyRes {
	cmd := &OmniGetPropertyCmd{PropertyId: int(req.PropertyId)}
	return c.SendCmd(cmd)
}

// omniGetProperty Returns details for about the tokens or smart property to lookup.
func OmniGetProperty(req *toolrpc.OmniGetPropertyReq, c *rpcclient.Client) (*toolrpc.OmniGetPropertyRes, error) {
	return omniGetPropertyAsync(req, c).Receive()
}

func init() {
	err := btcjson.RegisterCmd("omni_getproperty", &OmniGetPropertyCmd{}, btcjson.UFWalletOnly)
	if err != nil {
		panic(err)
	}
}

type OmniListpropertiesCmd struct {
}
type futureOmniListproperties chan *rpcclient.Response

func (r futureOmniListproperties) Receive() (*toolrpc.ListPropertiesRes, error) {
	res, err := rpcclient.ReceiveFuture(r)
	if err != nil {
		return nil, err
	}
	// take care of the special case where the output has been spent already
	// it should return the string "null"
	if string(res) == "null" {
		return nil, nil
	}
	// Unmarshal result as an omniGetProperty result object.
	var items []*toolrpc.Property
	err = json.Unmarshal(res, &items)
	if err != nil {
		return nil, err
	}

	return &toolrpc.ListPropertiesRes{Items: items}, nil
}
func OmniListpropertiesAsync(req *emptypb.Empty, c *rpcclient.Client) futureOmniListproperties {
	cmd := &OmniListpropertiesCmd{}
	return c.SendCmd(cmd)
}
func OmniListproperties(req *emptypb.Empty, c *rpcclient.Client) (*toolrpc.ListPropertiesRes, error) {
	return OmniListpropertiesAsync(req, c).Receive()
}
func init() {
	err := btcjson.RegisterCmd("omni_listproperties", &OmniListpropertiesCmd{}, btcjson.UFWalletOnly)
	if err != nil {
		panic(err)
	}
}

type OmniCreatePropertyCmd struct {
	Fromaddress string
	Ecosystem   int
	TokenType   int
	Previousid  int
	Category    string
	Subcategory string
	Name        string
	Url         string
	Data        string
	Amount      string
}
type futureOmniCreatePropertyRes chan *rpcclient.Response

func (r futureOmniCreatePropertyRes) Receive() (*toolrpc.CreatePropertyRes, error) {
	res, err := rpcclient.ReceiveFuture(r)
	if err != nil {
		return nil, err
	}

	// take care of the special case where the output has been spent already
	// it should return the string "null"
	if string(res) == "null" {
		return nil, nil
	}
	log.Println(string(res), err)
	return &toolrpc.CreatePropertyRes{Hash: string(res)}, nil
}

//omni_sendissuancefixed "2N9DZN7HX532Wbm8YqLz28RrMbETjM2aXFL" 2 2 0 "t1" "" "ftoken" "baidu.com" "" 10000000000
func OmniCreatePropertyAsync(req *toolrpc.CreatePropertyReq, c *rpcclient.Client) futureOmniCreatePropertyRes {
	cmd := &OmniCreatePropertyCmd{Fromaddress: req.Fromaddress,
		Ecosystem:   2,
		TokenType:   2,
		Previousid:  0,
		Category:    "",
		Subcategory: "",
		Name:        req.Name,
		Url:         "",
		Data:        "",
		Amount:      req.Amount}
	return c.SendCmd(cmd)
}

// OmniGetbalance Returns the token balance for a given address and property.
func OmniCreateProperty(req *toolrpc.CreatePropertyReq, c *rpcclient.Client) (*toolrpc.CreatePropertyRes, error) {
	return OmniCreatePropertyAsync(req, c).Receive()
}

func init() {
	err := btcjson.RegisterCmd("omni_sendissuancefixed", (*OmniCreatePropertyCmd)(nil), btcjson.UFWalletOnly)
	if err != nil {
		panic(err)
	}
}
