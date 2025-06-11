package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
)

const (
	rsaKeyBits        = 2048
	certValidityYears = 1
	orgName           = "GRPC_Plugins"
)

type PrivateCA struct {
	PrivateKey *rsa.PrivateKey
	Cert       *x509.Certificate
	CertBytes  []byte
}

func GeneratePrivateCA() (*PrivateCA, error) {
	logger := slog.Default().With("component", "transport")
	logger.Debug("generating private CA")

	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		logger.Error("failed to generate private key", "error", err)
		return nil, errors.Wrap(err, "failed to generate CA private key")
	}

	// Prepare certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"GRPC_Plugins"},
		},
		NotBefore:             time.Now().Add(-time.Second),
		NotAfter:              time.Now().AddDate(certValidityYears, 0, 0), // Valid for certValidityYears
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Self-sign the CA certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		logger.Error("failed to create CA certificate", "error", err)
		return nil, errors.Wrap(err, "failed to create CA certificate")
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		logger.Error("failed to parse CA certificate", "error", err)
		return nil, errors.Wrap(err, "failed to parse CA certificate")
	}

	logger.Debug("private CA generated successfully")
	return &PrivateCA{
		PrivateKey: privateKey,
		Cert:       cert,
		CertBytes:  certBytes,
	}, nil
}

type KeyAndCert struct {
	CN          string
	Key         *rsa.PrivateKey
	CACert      *x509.Certificate
	CACertBytes []byte
	Cert        *x509.Certificate
	CertBytes   []byte
}

func (k *KeyAndCert) GetTLSConfig() (*tls.Config, error) {
	logger := slog.Default().With("component", "transport", "cn", k.CN)
	logger.Debug("creating TLS config")

	certPool := x509.NewCertPool()
	certPool.AddCert(k.CACert)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS13,
		MaxVersion:         tls.VersionTLS13,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{k.Cert.Raw},
				PrivateKey:  k.Key,
				Leaf:        k.Cert,
			}},
		RootCAs:   certPool,
		ClientCAs: certPool,
	}

	logger.Debug("TLS config created successfully")
	return tlsConfig, nil
}

type KeyAndCertSerialized struct {
	CertBytes   string `json:"cert_bytes"`
	CACertBytes string `json:"ca_cert_bytes"`
	PrivateKey  string `json:"private_key_pkcs1"`
	CN          string `json:"cn"`
}

func (k *KeyAndCert) Serialize() ([]byte, error) {
	logger := slog.Default().With("component", "transport", "cn", k.CN)
	logger.Debug("serializing key and cert")

	toSerialize := KeyAndCertSerialized{
		CertBytes:   base64.StdEncoding.EncodeToString(k.Cert.Raw),
		CACertBytes: base64.StdEncoding.EncodeToString(k.CACert.Raw),
		PrivateKey:  base64.StdEncoding.EncodeToString(x509.MarshalPKCS1PrivateKey(k.Key)),
		CN:          k.CN,
	}

	jsonBytes, err := json.Marshal(toSerialize)
	if err != nil {
		logger.Error("failed to marshal key and cert", "error", err)
		return nil, errors.Wrap(err, "failed to marshal key and cert to JSON")
	}

	logger.Debug("key and cert serialized successfully")
	return jsonBytes, nil
}

func DeserializeKeyAndCert(data []byte) (*KeyAndCert, error) {
	logger := slog.Default().With("component", "transport")
	logger.Debug("deserializing key and cert")

	var toDeserialize KeyAndCertSerialized
	err := json.Unmarshal(data, &toDeserialize)
	if err != nil {
		logger.Error("failed to unmarshal key and cert data", "error", err)
		return nil, errors.Wrap(err, "failed to unmarshal key and cert data from JSON")
	}

	k := &KeyAndCert{}
	k.CertBytes, err = base64.StdEncoding.DecodeString(toDeserialize.CertBytes)
	if err != nil {
		logger.Error("failed to decode cert bytes", "error", err)
		return nil, errors.Wrap(err, "failed to decode certificate bytes from base64")
	}
	k.CACertBytes, err = base64.StdEncoding.DecodeString(toDeserialize.CACertBytes)
	if err != nil {
		logger.Error("failed to decode CA cert bytes", "error", err)
		return nil, errors.Wrap(err, "failed to decode CA certificate bytes from base64")
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(toDeserialize.PrivateKey)
	if err != nil {
		logger.Error("failed to decode private key", "error", err)
		return nil, errors.Wrap(err, "failed to decode private key from base64")
	}
	k.Key, err = x509.ParsePKCS1PrivateKey(privateKeyBytes)
	if err != nil {
		logger.Error("failed to parse private key", "error", err)
		return nil, errors.Wrap(err, "failed to parse PKCS1 private key")
	}

	k.CN = toDeserialize.CN

	cert, err := x509.ParseCertificate(k.CertBytes)
	if err != nil {
		logger.Error("failed to parse certificate", "error", err)
		return nil, errors.Wrap(err, "failed to parse certificate")
	}
	k.Cert = cert

	caCert, err := x509.ParseCertificate(k.CACertBytes)
	if err != nil {
		logger.Error("failed to parse CA certificate", "error", err)
		return nil, errors.Wrap(err, "failed to parse CA certificate")
	}
	k.CACert = caCert

	logger.Debug("key and cert deserialized successfully", "cn", k.CN)
	return k, nil
}

type Role string

const (
	RoleServer Role = "server"
	RoleClient Role = "client"
)

func GenerateKeyAndCertFromCA(ca *PrivateCA, subject string, role Role) (*KeyAndCert, error) {
	logger := slog.Default().With("component", "transport", "subject", subject, "role", role)
	logger.Debug("generating key and cert from CA")

	if ca == nil {
		logger.Error("CA cannot be nil")
		return nil, errors.New("CA cannot be nil")
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		logger.Error("failed to generate private key", "error", err)
		return nil, errors.Wrapf(err, "failed to generate private key for %s", subject)
	}

	var usage []x509.ExtKeyUsage
	switch role {
	case "server":
		usage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	case "client":
		usage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	default:
		return nil, errors.Errorf("invalid role %q, must be 'server' or 'client'", role)
	}

	sn, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		logger.Error("failed to generate serial number", "error", err)
		return nil, errors.Wrapf(err, "failed to generate serial number for %s", subject)
	}

	template := &x509.Certificate{
		SerialNumber: sn,
		Subject: pkix.Name{
			CommonName:   subject,
			Organization: []string{"GRPC_Plugins"},
		},
		NotBefore:   time.Now().Add(-time.Second),
		NotAfter:    time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: usage,
		// TODO: Is this a security concern?
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// Sign the server certificate with our CA
	certBytes, err := x509.CreateCertificate(rand.Reader, template, ca.Cert, &serverKey.PublicKey, ca.PrivateKey)
	if err != nil {
		logger.Error("failed to create certificate", "error", err)
		return nil, errors.Wrapf(err, "failed to create certificate for %s", subject)
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		logger.Error("failed to parse certificate", "error", err)
		return nil, errors.Wrapf(err, "failed to parse certificate for %s", subject)
	}

	logger.Debug("key and cert generated successfully")
	return &KeyAndCert{
		CN:          subject,
		Key:         serverKey,
		CACert:      ca.Cert,
		CACertBytes: ca.CertBytes,
		Cert:        cert,
		CertBytes:   certBytes,
	}, nil
}
