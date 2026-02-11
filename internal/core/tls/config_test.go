package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// generateTestCert creates a self-signed cert and key pair in the given directory.
func generateTestCert(t *testing.T, dir string) (certPath, keyPath string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating cert: %v", err)
	}

	certPath = filepath.Join(dir, "cert.pem")
	certFile, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("creating cert file: %v", err)
	}
	pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	certFile.Close()

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshaling key: %v", err)
	}

	keyPath = filepath.Join(dir, "key.pem")
	keyFile, err := os.Create(keyPath)
	if err != nil {
		t.Fatalf("creating key file: %v", err)
	}
	pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	keyFile.Close()

	return certPath, keyPath
}

func TestBuildTLSConfig_NilConfig(t *testing.T) {
	var cfg *Config
	tlsCfg, err := cfg.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tlsCfg != nil {
		t.Error("expected nil tls config for nil config")
	}
}

func TestBuildTLSConfig_InsecureSkipVerify(t *testing.T) {
	cfg := &Config{InsecureSkipVerify: true}
	tlsCfg, err := cfg.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tlsCfg.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true")
	}
}

func TestBuildTLSConfig_ClientCert(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCert(t, dir)

	cfg := &Config{
		CertFile: certPath,
		KeyFile:  keyPath,
	}
	tlsCfg, err := cfg.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tlsCfg.Certificates) != 1 {
		t.Errorf("expected 1 certificate, got %d", len(tlsCfg.Certificates))
	}
}

func TestBuildTLSConfig_InvalidCert(t *testing.T) {
	cfg := &Config{
		CertFile: "/nonexistent/cert.pem",
		KeyFile:  "/nonexistent/key.pem",
	}
	_, err := cfg.BuildTLSConfig()
	if err == nil {
		t.Error("expected error for invalid cert paths")
	}
}

func TestBuildTLSConfig_CAFile(t *testing.T) {
	dir := t.TempDir()
	certPath, _ := generateTestCert(t, dir)

	cfg := &Config{CAFile: certPath}
	tlsCfg, err := cfg.BuildTLSConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tlsCfg.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
}

func TestBuildTLSConfig_InvalidCAFile(t *testing.T) {
	cfg := &Config{CAFile: "/nonexistent/ca.pem"}
	_, err := cfg.BuildTLSConfig()
	if err == nil {
		t.Error("expected error for invalid CA path")
	}
}

func TestBuildTLSConfig_InvalidCAPEM(t *testing.T) {
	dir := t.TempDir()
	badCA := filepath.Join(dir, "bad-ca.pem")
	os.WriteFile(badCA, []byte("not a valid PEM"), 0644)

	cfg := &Config{CAFile: badCA}
	_, err := cfg.BuildTLSConfig()
	if err == nil {
		t.Error("expected error for invalid CA PEM content")
	}
}

func TestIsEmpty(t *testing.T) {
	var nilCfg *Config
	if !nilCfg.IsEmpty() {
		t.Error("nil config should be empty")
	}

	emptyCfg := &Config{}
	if !emptyCfg.IsEmpty() {
		t.Error("zero-value config should be empty")
	}

	nonEmpty := &Config{InsecureSkipVerify: true}
	if nonEmpty.IsEmpty() {
		t.Error("config with InsecureSkipVerify should not be empty")
	}

	certCfg := &Config{CertFile: "cert.pem", KeyFile: "key.pem"}
	if certCfg.IsEmpty() {
		t.Error("config with cert files should not be empty")
	}
}
