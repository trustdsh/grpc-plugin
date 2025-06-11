package transport

import (
	"log/slog"

	"github.com/pkg/errors"
	"github.com/trustdsh/grpc-plugin/pkgs/config"
)

type TransportGenerator struct {
	ca  *PrivateCA
	cfg *config.TLSConfig
}

func NewTransportGenerator(cfg *config.TLSConfig) (*TransportGenerator, error) {
	logger := slog.Default().With("component", "transport_generator")
	logger.Debug("creating new transport generator")

	t := &TransportGenerator{
		cfg: cfg,
	}

	if cfg.UseCustomTLS {
		logger.Error("custom TLS is not supported")
		return nil, errors.New("custom TLS is not supported yet")
	}

	ca, err := GeneratePrivateCA()
	if err != nil {
		logger.Error("failed to generate private CA", "error", err)
		return nil, errors.Wrap(err, "failed to generate private CA")
	}
	t.ca = ca

	logger.Debug("transport generator created successfully")
	return t, nil
}

func (t *TransportGenerator) GenerateKeyAndCert(subject string, role Role) (*KeyAndCert, error) {
	logger := slog.Default().With("component", "transport_generator", "subject", subject, "role", role)
	logger.Debug("generating key and cert")

	// Validate role parameter
	if role != RoleServer && role != RoleClient {
		return nil, errors.Errorf("invalid role: %s, must be %s or %s", role, RoleServer, RoleClient)
	}

	keyAndCert, err := GenerateKeyAndCertFromCA(t.ca, subject, role)
	if err != nil {
		logger.Error("failed to generate key and cert", "error", err)
		return nil, errors.Wrapf(err, "failed to generate key and cert for %s with role %s", subject, role)
	}

	logger.Debug("key and cert generated successfully")
	return keyAndCert, nil
}
