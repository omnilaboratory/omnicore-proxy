package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/btcsuite/btcd/btcjson"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var btcHost string
var rpcUser string
var rpcPasswd string

func main() {
	flag.StringVar(&btcHost, "btc_host", "http://43.138.107.248:8332", "http://ip:port")
	flag.StringVar(&rpcUser, "rpc_user", "omniwallet", "")
	flag.StringVar(&rpcPasswd, "rpc_passwd", "cB3]iL2@eZ1?cB2?", "")
	flag.Parse()
	http.HandleFunc("/", mainHttpHandler)
	log.Println("server start at :18332")
	err := http.ListenAndServe(":18332", http.DefaultServeMux)
	if err != nil {
		log.Println(err)
	}
}

var whiteListMethods = map[string]bool{
	"getinfo":            true,
	"getblockchaininfo":  true,
	"gettransaction":     true,
	"help":               true,
	"getbestblock":       true,
	"getbestblockhash":   true,
	"getblockhash":       true,
	"getblockheader":     true,
	"getblockstats":      true,
	"getdifficulty":      true,
	"sendrawtransaction": true,
	// omni
	"getaddressbalance": true,
}

// maxRequestSize specifies the maximum number of bytes in the request body
// that may be read from a client.  This is currently limited to 4MB.
const maxRequestSize = 1024 * 1024 * 4

func mainHttpHandler(w http.ResponseWriter, r *http.Request) {
	body := http.MaxBytesReader(w, r.Body, maxRequestSize)
	rpcRequest, err := ioutil.ReadAll(body)
	if err != nil {
		// TODO: what if the underlying reader errored?
		http.Error(w, "413 Request Too Large.",
			http.StatusRequestEntityTooLarge)
		return
	}

	// First check whether wallet has a handler for this request's method.
	// If unfound, the request is sent to the chain server for further
	// processing.  While checking the methods, disallow authenticate
	// requests, as they are invalid for HTTP POST clients.
	var req btcjson.Request
	err = json.Unmarshal(rpcRequest, &req)
	if err != nil {
		resp, err := btcjson.MarshalResponse(
			btcjson.RpcVersion1, req.ID, nil,
			btcjson.ErrRPCInvalidRequest,
		)
		if err != nil {

			log.Printf("Unable to marshal response: %v", err)
			http.Error(w, "500 Internal Server Error",
				http.StatusInternalServerError)
			return
		}
		_, err = w.Write(resp)
		if err != nil {
			log.Printf("Cannot write invalid request request to "+
				"client: %v", err)
		}
		return
	}
	if req.Method == "" {
		log.Printf("no method")
		http.Error(w, "500 Internal Server Error",
			http.StatusInternalServerError)
		return
	}
	// Method may not be empty.
	if whiteListMethods[req.Method] || strings.HasPrefix(req.Method, "omni_get") || strings.HasPrefix(req.Method, "omni_list") {
		res, err := rawRequest(&req)
		if err != nil {
			log.Printf("rawRequest to btcooind err %v ", err)
			http.Error(w, "500 Internal Server Error",
				http.StatusInternalServerError)
			return
		} else {
			_, err = w.Write(res)
			if err != nil {
				log.Printf("Unable to respond to client: %v", err)
			}
		}

	} else {
		log.Printf("method %v disabled", req.Method)
		http.Error(w, "500 Internal Server Error",
			http.StatusInternalServerError)
		return
	}
}

var reqid uint64

func nextID() uint64 {
	return atomic.AddUint64(&reqid, 1)
}

func rawRequest(req *btcjson.Request) ([]byte, error) {
	// Create a raw JSON-RPC request using the provided method and params
	// and marshal it.  This is done rather than using the SendCmd function
	// since that relies on marshalling registered btcjson commands rather
	// than custom commands.
	req.ID = nextID()
	req.Jsonrpc = btcjson.RpcVersion1

	var (
		err, lastErr error
		backoff      time.Duration
		httpResponse *http.Response
	)

	tries := 10
	for i := 0; i < tries; i++ {
		var httpReq *http.Request
		bs, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}
		bodyReader := bytes.NewReader(bs)
		httpReq, err = http.NewRequest("POST", btcHost, bodyReader)
		if err != nil {
			return nil, err
		}
		httpReq.Close = true
		httpReq.Header.Set("Content-Type", "application/json")

		// Configure basic access authorization.
		httpReq.SetBasicAuth(rpcUser, rpcPasswd)

		httpResponse, err = http.DefaultClient.Do(httpReq)

		// Quit the retry loop on success or if we can't retry anymore.
		if err == nil || i == tries-1 {
			break
		}

		// Save the last error for the case where we backoff further,
		// retry and get an invalid response but no error. If this
		// happens the saved last error will be used to enrich the error
		// message that we pass back to the caller.
		lastErr = err

		// Backoff sleep otherwise.
		backoff = 500 * time.Millisecond * time.Duration(i+1)
		if backoff > time.Minute {
			backoff = time.Minute
		}
		log.Printf("Failed command [%s] with id %d attempt %d."+
			" Retrying in %v... \n", req.Method, req.ID,
			i, backoff)

		select {
		case <-time.After(backoff):
		}
	}
	if err != nil {
		return nil, err
	}

	// We still want to return an error if for any reason the respone
	// remains empty.
	if httpResponse == nil {
		return nil, fmt.Errorf("invalid http POST response (nil), "+
			"method: %s, id: %d, last error=%v",
			req.Method, req.ID, lastErr)
	}

	// Read the raw bytes and close the response.
	respBytes, err := ioutil.ReadAll(httpResponse.Body)
	httpResponse.Body.Close()
	if err != nil {
		err = fmt.Errorf("error reading json reply: %v", err)
	}
	return respBytes, err
}
