package test_utils

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"strings"
	"time"
)

func GetNextFileDescriptor() uintptr {
	handle, _ := os.Open("/dev/null")
	fd := handle.Fd()
	handle.Close()
	return fd
}

func PublicKey(priv any) any {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	case ed25519.PrivateKey:
		return k.Public().(ed25519.PublicKey)
	default:
		return nil
	}
}

func GenerateX509() (certFile string, keyFile string, cleanup func(), err error) {
	var priv *ecdsa.PrivateKey
	priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return certFile, keyFile, func() {}, err
	}
	keyUsage := x509.KeyUsageDigitalSignature
	// if _, isRSA := priv.(*rsa.PrivateKey); isRSA {
	// 	keyUsage |= x509.KeyUsageKeyEncipherment
	// }

	notBefore := time.Now().Add(-time.Hour)
	notAfter := notBefore.Add(time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"GoLang Acme Co"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              keyUsage,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split("aws-relay.test,127.0.0.1", ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	// if *isCA {
	template.IsCA = true
	template.KeyUsage |= x509.KeyUsageCertSign
	// }

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, PublicKey(priv), priv)
	if err != nil {
		return certFile, keyFile, func() {}, err
	}

	certOut, err := os.CreateTemp("./", "cert-*.pem")
	if err != nil {
		return certFile, keyFile, func() {}, err
	}
	// var certBytes bytes.Buffer
	// certOut := bufio.NewWriter(&certBytes)
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return certFile, keyFile, func() {}, err
	}
	certOut.Close()
	certFile = certOut.Name()

	keyOut, err := os.CreateTemp("./", "key-*.pem")
	if err != nil {
		return certFile, keyFile, func() {}, err
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return certFile, keyFile, func() {}, err
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return certFile, keyFile, func() {}, err
	}
	keyOut.Close()
	keyFile = keyOut.Name()

	return certFile, keyFile, func() {
		os.Remove(certFile)
		os.Remove(keyFile)
	}, nil
}
