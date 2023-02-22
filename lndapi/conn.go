package lndapi

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lightningnetwork/lnd/lnrpc/routerrpc"
	"github.com/lightningnetwork/lnd/macaroons"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/macaroon.v2"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

// profileEntry is a struct that represents all settings for one specific
// profile.
type profileEntry struct {
	Name        string       `json:"name"`
	RPCServer   string       `json:"rpcserver"`
	LndDir      string       `json:"lnddir"`
	Chain       string       `json:"chain"`
	Network     string       `json:"network"`
	NoMacaroons bool         `json:"no-macaroons,omitempty"`
	TLSCert     string       `json:"tlscert"`
	Macaroons   *macaroonJar `json:"macaroons"`
}

// cert returns the profile's TLS certificate as a x509 certificate pool.
func (e *profileEntry) cert() (*x509.CertPool, error) {
	if e.TLSCert == "" {
		return nil, nil
	}

	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM([]byte(e.TLSCert)) {
		return nil, fmt.Errorf("credentials: failed to append " +
			"certificate")
	}
	return cp, nil
}

// profileFromContext creates an ephemeral profile entry from the global options
// set in the CLI context.
func profileFromContext(rpcAddress string, netType string, lndDir string) (
	*profileEntry, error) {

	macPath := filepath.Join(
		lndDir, defaultDataDir, defaultChainSubDir, "bitcoin",
		netType, defaultMacaroonFilename,
	)
	tlsCertPath := filepath.Join(lndDir, defaultTLSCertFilename)

	// Load the certificate file now, if specified. We store it as plain PEM
	// directly.
	var tlsCert []byte
	if tlsCertPath != "" {
		var err error
		tlsCert, err = ioutil.ReadFile(tlsCertPath)
		if err != nil {
			return nil, fmt.Errorf("could not load TLS cert file: %v", err)
		}
	}

	entry := &profileEntry{
		RPCServer:   rpcAddress,
		LndDir:      lndDir,
		Chain:       "bitcoin",
		Network:     netType,
		NoMacaroons: false,
		TLSCert:     string(tlsCert),
	}

	// Now load and possibly encrypt the macaroon file.
	macBytes, err := ioutil.ReadFile(macPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read macaroon path (check "+
			"the network setting!): %v", err)
	}
	mac := &macaroon.Macaroon{}
	if err = mac.UnmarshalBinary(macBytes); err != nil {
		return nil, fmt.Errorf("unable to decode macaroon: %v", err)
	}

	var pw []byte
	//if store {
	//	// Read a password from the terminal. If it's empty, we won't
	//	// encrypt the macaroon and store it plaintext.
	//	pw, err = capturePassword(
	//		"Enter password to encrypt macaroon with or leave "+
	//			"blank to store in plaintext: ", true,
	//		walletunlocker.ValidatePassword,
	//	)
	//	if err != nil {
	//		return nil, fmt.Errorf("unable to get encryption "+
	//			"password: %v", err)
	//	}
	//}
	macEntry := &macaroonEntry{}
	if err = macEntry.storeMacaroon(mac, pw); err != nil {
		return nil, fmt.Errorf("unable to store macaroon: %v", err)
	}

	// We determine the name of the macaroon from the file itself but cut
	// off the ".macaroon" at the end.
	macEntry.Name = path.Base(macPath)
	if path.Ext(macEntry.Name) == "macaroon" {
		macEntry.Name = strings.TrimSuffix(macEntry.Name, ".macaroon")
	}

	// Now that we have the macaroon jar as well, let's return the entry
	// with all the values populated.
	entry.Macaroons = &macaroonJar{
		Default: macEntry.Name,
		Timeout: 60,
		IP:      "",
		Jar:     []*macaroonEntry{macEntry},
	}

	return entry, nil

}

func cert(tlsFilePath string) (*x509.CertPool, error) {
	cp := x509.NewCertPool()
	if !cp.AppendCertsFromPEM([]byte(tlsFilePath)) {
		return nil, fmt.Errorf("credentials: failed to append " +
			"certificate")
	}
	return cp, nil
}

const (
	defaultDataDir          = "data"
	defaultChainSubDir      = "chain"
	defaultTLSCertFilename  = "tls.cert"
	defaultMacaroonFilename = "admin.macaroon"
	defaultRPCPort          = "10009"
	defaultRPCHostPort      = "localhost:" + defaultRPCPort
)

var lndClients = map[string]lnrpc.LightningClient{}
var localLnSync sync.Mutex

func GetLndClient(rpcAddress string, netType, lnDir string) (lnrpc.LightningClient, error) {
	localLnSync.Lock()
	defer localLnSync.Unlock()
	cli, ok := lndClients[rpcAddress]
	if ok {
		return cli, nil
	}

	conn, err := getClientConn(rpcAddress, netType, lnDir)
	if err != nil {
		return nil, err
	}
	//cleanUp := func() {
	//	conn.Close()
	//}
	cli = lnrpc.NewLightningClient(conn)
	lndClients[rpcAddress] = cli
	return cli, nil
}

var lnRouterClients = map[string]routerrpc.RouterClient{}
var localRouterSync sync.Mutex

func GetRouterClient(rpcAddress string, netType, lnDir string) (routerrpc.RouterClient, error) {
	localRouterSync.Lock()
	defer localRouterSync.Unlock()
	cli, ok := lnRouterClients[rpcAddress]
	if ok {
		return cli, nil
	}

	conn, err := getClientConn(rpcAddress, netType, lnDir)
	if err != nil {
		return nil, err
	}
	//cleanUp := func() {
	//	conn.Close()
	//}
	cli = routerrpc.NewRouterClient(conn)
	lnRouterClients[rpcAddress] = cli
	return cli, nil
}

func getClientConn(rpcAddress string, netType, lnDir string) (*grpc.ClientConn, error) {
	// First, we'll get the selected stored profile or an ephemeral one
	// created from the global options in the CLI context.

	profile, err := profileFromContext(rpcAddress, netType, lnDir)
	if err != nil {
		return nil, fmt.Errorf("profileFromContext err: %v", err)
	}

	// Load the specified TLS certificate.
	certPool, err := profile.cert()
	if err != nil {
		return nil, fmt.Errorf("could not create cert pool: %v", err)
	}

	// Build transport credentials from the certificate pool. If there is no
	// certificate pool, we expect the server to use a non-self-signed
	// certificate such as a certificate obtained from Let's Encrypt.
	var creds credentials.TransportCredentials
	if certPool != nil {
		creds = credentials.NewClientTLSFromCert(certPool, "")
	} else {
		// Fallback to the system pool. Using an empty tls config is an
		// alternative to x509.SystemCertPool(). That call is not
		// supported on Windows.
		creds = credentials.NewTLS(&tls.Config{})
	}

	// Create a dial options array.
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	// Only process macaroon credentials if --no-macaroons isn't set and
	// if we're not skipping macaroon processing.
	if !profile.NoMacaroons {
		// Find out which macaroon to load.
		macName := profile.Macaroons.Default
		//if ctx.GlobalIsSet("macfromjar") {
		//	macName = ctx.GlobalString("macfromjar")
		//}
		var macEntry *macaroonEntry
		for _, entry := range profile.Macaroons.Jar {
			if entry.Name == macName {
				macEntry = entry
				break
			}
		}
		if macEntry == nil {
			return nil, fmt.Errorf("macaroon with name '%s' not found "+
				"in profile", macName)
		}

		// Get and possibly decrypt the specified macaroon.
		//
		// TODO(guggero): Make it possible to cache the password so we
		// don't need to ask for it every time.
		mac, err := macEntry.loadMacaroon(nil)
		if err != nil {
			return nil, fmt.Errorf("could not load macaroon: %v", err)
		}

		macConstraints := []macaroons.Constraint{
			// We add a time-based constraint to prevent replay of the
			// macaroon. It's good for 60 seconds by default to make up for
			// any discrepancy between client and server clocks, but leaking
			// the macaroon before it becomes invalid makes it possible for
			// an attacker to reuse the macaroon. In addition, the validity
			// time of the macaroon is extended by the time the server clock
			// is behind the client clock, or shortened by the time the
			// server clock is ahead of the client clock (or invalid
			// altogether if, in the latter case, this time is more than 60
			// seconds).
			// TODO(aakselrod): add better anti-replay protection.
			macaroons.TimeoutConstraint(profile.Macaroons.Timeout),

			// Lock macaroon down to a specific IP address.
			macaroons.IPLockConstraint(profile.Macaroons.IP),

			// ... Add more constraints if needed.
		}

		// Apply constraints to the macaroon.
		constrainedMac, err := macaroons.AddConstraints(
			mac, macConstraints...,
		)
		if err != nil {
			return nil, err
		}

		// Now we append the macaroon credentials to the dial options.
		cred, err := macaroons.NewMacaroonCredential(constrainedMac)
		if err != nil {
			return nil, fmt.Errorf("error cloning mac: %v", err)
		}
		opts = append(opts, grpc.WithPerRPCCredentials(cred))
	}

	// We need to use a custom dialer so we can also connect to unix sockets
	// and not just TCP addresses.
	//genericDialer := lncfg.ClientAddressDialer(defaultRPCPort)
	//opts = append(opts, grpc.WithContextDialer(genericDialer))
	//opts = append(opts, grpc.WithDefaultCallOptions(maxMsgRecvSize))

	conn, err := grpc.Dial(profile.RPCServer, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to RPC server: %v", err)
	}

	return conn, nil
}
