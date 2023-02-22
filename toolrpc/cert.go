package toolrpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"math/big"
)

func getCertPub(ctx context.Context) (string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "getCertPub no peer found")
	}

	tlsAuth, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "getCertPub unexpected peer transport credentials")
	}
	if len(tlsAuth.State.PeerCertificates) == 0 {
		return "", status.Error(codes.Unauthenticated, "getCertPub client miss tls cert")
	}
	pubStr, err := pubKeyStr(tlsAuth.State.PeerCertificates[0].PublicKey)
	if err != nil {
		return "", status.Error(codes.Unauthenticated, err.Error())
	}
	return pubStr, nil
}

type pkcs1PublicKey struct {
	N *big.Int
	E int
}

func pubKeyStr(pub interface{}) (string, error) {
	publicKeyBytes := []byte{}
	var err error
	switch pub := pub.(type) {
	case *rsa.PublicKey:
		publicKeyBytes, err = asn1.Marshal(pkcs1PublicKey{
			N: pub.N,
			E: pub.E,
		})
		if err != nil {
			return "", errors.New("error rsa.PublicKey")
		}
	case *ecdsa.PublicKey:
		publicKeyBytes = elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	case ed25519.PublicKey:
		publicKeyBytes = pub
	default:
		return "", fmt.Errorf("x509: unsupported public key type: %T", pub)
	}
	return hex.EncodeToString(publicKeyBytes), nil
}
