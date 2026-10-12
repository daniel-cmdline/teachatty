package tlstrust

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// startServer runs an HTTPS test server with the certificate stored in certDir
func startServer(t *testing.T, certDir string) (*httptest.Server, string) {
	t.Helper()
	cfg, fp, err := ServerConfig(certDir)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.TLS = cfg
	srv.StartTLS()
	t.Cleanup(srv.Close)
	return srv, fp
}

// get makes one HTTPS request through ClientConfig, naming the server "chat:7347"
// whatever its real address is, so two different servers can pose as the same name
func get(srv *httptest.Server, knownFile string, notice *bytes.Buffer) error {
	tr := &http.Transport{TLSClientConfig: ClientConfig("chat:7347", knownFile, notice)}
	defer tr.CloseIdleConnections()
	resp, err := (&http.Client{Transport: tr}).Get(srv.URL)
	if err == nil {
		resp.Body.Close()
	}
	return err
}

func TestCertificateIsGeneratedOnceAndReused(t *testing.T) {
	dir := t.TempDir()
	_, fp1, err := ServerConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, fp2, err := ServerConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fp1 != fp2 {
		t.Fatalf("fingerprint changed between runs: %s then %s", fp1, fp2)
	}
	if !strings.HasPrefix(fp1, "SHA256:") {
		t.Fatalf("fingerprint %q should start with SHA256:", fp1)
	}
	info, err := os.Stat(filepath.Join(dir, "server.key"))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("server.key has mode %o, want 600", mode)
	}
}

func TestTrustOnFirstUseThenPinned(t *testing.T) {
	srv, fp := startServer(t, t.TempDir())
	known := filepath.Join(t.TempDir(), "known_servers")

	// first connection: accepted, remembered, and announced
	var notice bytes.Buffer
	if err := get(srv, known, &notice); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notice.String(), "trusting new server chat:7347") {
		t.Fatalf("expected a first-use notice, got %q", notice.String())
	}
	if strings.Contains(notice.String(), "warning") {
		t.Fatalf("teachatty's own self-signed certificate shouldn't warn, got %q", notice.String())
	}
	saved, _ := os.ReadFile(known)
	if string(saved) != "chat:7347 "+fp+"\n" {
		t.Fatalf("known_servers has %q", saved)
	}

	// second connection: accepted silently, nothing new written
	notice.Reset()
	if err := get(srv, known, &notice); err != nil {
		t.Fatal(err)
	}
	if notice.Len() != 0 {
		t.Fatalf("expected no notice on the second connection, got %q", notice.String())
	}
	again, _ := os.ReadFile(known)
	if !bytes.Equal(saved, again) {
		t.Fatalf("known_servers changed: %q", again)
	}
}

func TestChangedServerIsRefused(t *testing.T) {
	known := filepath.Join(t.TempDir(), "known_servers")
	var notice bytes.Buffer

	real, _ := startServer(t, t.TempDir())
	if err := get(real, known, &notice); err != nil {
		t.Fatal(err)
	}

	// another server (different certificate) answering under the same name
	impostor, _ := startServer(t, t.TempDir())
	err := get(impostor, known, &notice)
	if err == nil || !strings.Contains(err.Error(), "has changed") {
		t.Fatalf("expected the connection to be refused as changed, got %v", err)
	}
}

// testCA is a certificate authority made up for the tests, standing in for Let's Encrypt
type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pool *x509.CertPool
}

func newTestCA(t *testing.T) *testCA {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return &testCA{cert, key, pool}
}

// issue writes a certificate for name, signed by the CA, plus its key, into dir
func (ca *testCA) issue(t *testing.T, dir, name string) (certFile, keyFile string) {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: name},
		DNSNames:     []string{name},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, _ := x509.MarshalPKCS8PrivateKey(key)
	certFile, keyFile = filepath.Join(dir, "fullchain.pem"), filepath.Join(dir, "privkey.pem")
	if err := writePEM(certFile, "CERTIFICATE", der, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writePEM(keyFile, "PRIVATE KEY", keyDER, 0o600); err != nil {
		t.Fatal(err)
	}
	return certFile, keyFile
}

// serveProvided runs an HTTPS test server with the administrator's certificate files and
// returns its URL. It wraps the listener with tls.NewListener, like the real server does:
// httptest's own StartTLS would add its example certificate next to ours and serve that.
func serveProvided(t *testing.T, certFile, keyFile string) string {
	t.Helper()
	cfg, _, err := providedConfig(certFile, keyFile, io.Discard, nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Listener = tls.NewListener(srv.Listener, cfg)
	srv.Start()
	t.Cleanup(srv.Close)
	return "https://" + srv.Listener.Addr().String()
}

// getWith makes one HTTPS request to url with the given client config
func getWith(url string, cfg *tls.Config) error {
	tr := &http.Transport{TLSClientConfig: cfg}
	defer tr.CloseIdleConnections()
	resp, err := (&http.Client{Transport: tr}).Get(url)
	if err == nil {
		resp.Body.Close()
	}
	return err
}

func TestAuthoritySignedCertificateIsNotPinned(t *testing.T) {
	ca := newTestCA(t)
	certFile, keyFile := ca.issue(t, t.TempDir(), "chat")
	srv := serveProvided(t, certFile, keyFile)
	known := filepath.Join(t.TempDir(), "known_servers")

	var notice bytes.Buffer
	if err := getWith(srv, clientConfig("chat:7347", known, &notice, ca.pool)); err != nil {
		t.Fatal(err)
	}
	if notice.Len() != 0 {
		t.Fatalf("a certificate from a trusted authority shouldn't go through first-use trust, got %q", notice.String())
	}
	if _, err := os.Stat(known); err == nil {
		t.Fatal("known_servers shouldn't be written for an authority-signed certificate")
	}

	// a renewal with a brand new key (what Let's Encrypt does) is still accepted
	certFile2, keyFile2 := ca.issue(t, t.TempDir(), "chat")
	srv2 := serveProvided(t, certFile2, keyFile2)
	if err := getWith(srv2, clientConfig("chat:7347", known, &notice, ca.pool)); err != nil {
		t.Fatalf("renewed certificate refused: %v", err)
	}
}

func TestAuthorityCertificateForAnotherNameFallsBackToFirstUse(t *testing.T) {
	ca := newTestCA(t)
	certFile, keyFile := ca.issue(t, t.TempDir(), "other-host")
	srv := serveProvided(t, certFile, keyFile)
	var notice bytes.Buffer
	if err := getWith(srv, clientConfig("chat:7347", filepath.Join(t.TempDir(), "known"), &notice, ca.pool)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(notice.String(), "trusting new server chat:7347") {
		t.Fatalf("expected first-use trust for a name mismatch, got %q", notice.String())
	}
	// and a warning: this one will be refused after the server renews it
	if !strings.Contains(notice.String(), `* warning: this certificate is issued by "Test CA" but didn't verify for chat`) {
		t.Fatalf("expected a warning about the unverified authority certificate, got %q", notice.String())
	}
}

func TestProvidedCertificateIsReloadedWhenFilesChange(t *testing.T) {
	ca := newTestCA(t)
	dir := t.TempDir()
	certFile, keyFile := ca.issue(t, dir, "chat")
	cfg, desc, err := providedConfig(certFile, keyFile, io.Discard, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(desc, "certificate for chat, issued by Test CA") {
		t.Fatalf("unexpected description %q", desc)
	}
	first, _ := cfg.GetCertificate(nil)

	// a renewal overwrites both files; make sure the new ones look newer on disk
	ca.issue(t, dir, "chat")
	later := time.Now().Add(time.Minute)
	os.Chtimes(certFile, later, later)
	os.Chtimes(keyFile, later, later)

	second, _ := cfg.GetCertificate(nil)
	if Fingerprint(first.Leaf) == Fingerprint(second.Leaf) {
		t.Fatal("the renewed certificate wasn't picked up")
	}
}

func TestCertAndKeyGoTogether(t *testing.T) {
	if _, _, err := Server("cert.pem", "", io.Discard); err == nil {
		t.Fatal("expected an error with --tls-cert but no --tls-key")
	}
}

func TestServerWarnsWhenTheChainDoesNotVerify(t *testing.T) {
	ca := newTestCA(t)
	certFile, keyFile := ca.issue(t, t.TempDir(), "chat")

	// a machine that trusts the authority: no warning
	var warn bytes.Buffer
	if _, _, err := providedConfig(certFile, keyFile, &warn, ca.pool); err != nil {
		t.Fatal(err)
	}
	if warn.Len() != 0 {
		t.Fatalf("expected no warning, got %q", warn.String())
	}

	// a machine that doesn't (like cert.pem without the intermediate): warning + hint
	if _, _, err := providedConfig(certFile, keyFile, &warn, x509.NewCertPool()); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"doesn't verify against this machine's trusted authorities", "use fullchain.pem, not cert.pem"} {
		if !strings.Contains(warn.String(), want) {
			t.Fatalf("warning should mention %q, got %q", want, warn.String())
		}
	}
}
