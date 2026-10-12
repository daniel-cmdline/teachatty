// Package tlstrust gives teachat encrypted connections without anyone managing
// certificates. The server generates its own self-signed certificate once and keeps
// using it. The client trusts a server the first time it connects, remembers its
// fingerprint, and from then on refuses any other certificate for that server:
// the same model SSH uses with known_hosts.
package tlstrust

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Dir returns teachat's config directory (~/.config/teachat on Linux), creating it.
// The server's certificate and the client's known_servers file live there.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "teachat")
	return dir, os.MkdirAll(dir, 0o700)
}

// ServerConfig loads the server's certificate from dir, generating it on the first
// run. It returns the TLS config and the certificate's fingerprint, which the server
// shows so people can check they're talking to the right machine.
func ServerConfig(dir string) (*tls.Config, string, error) {
	certFile := filepath.Join(dir, "server.crt")
	keyFile := filepath.Join(dir, "server.key")
	if _, err := os.Stat(certFile); errors.Is(err, fs.ErrNotExist) {
		if err := generate(certFile, keyFile); err != nil {
			return nil, "", fmt.Errorf("generating the server certificate: %w", err)
		}
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, "", err
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS13}
	return cfg, Fingerprint(cert.Leaf), nil
}

// generate creates a self-signed certificate valid for 10 years. Nobody checks the
// hostname or a certificate authority here: the client pins the fingerprint instead,
// so the certificate only has to be stable and unique to this server.
func generate(certFile, keyFile string) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "teachat-server"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return err
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	// the private key is readable only by its owner; the certificate is public anyway
	if err := writePEM(keyFile, "PRIVATE KEY", keyDER, 0o600); err != nil {
		return err
	}
	return writePEM(certFile, "CERTIFICATE", der, 0o644)
}

func writePEM(path, kind string, der []byte, mode os.FileMode) error {
	return os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: der}), mode)
}

// Fingerprint identifies a server, SSH style: "SHA256:" + base64 of the hash of the
// certificate's public key. Pinning the public key (not the whole certificate) means
// a certificate renewed with the same key keeps the same fingerprint.
func Fingerprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
}

// ClientConfig returns a TLS config for connecting to server (as typed in --server).
// A certificate from an authority this machine trusts is verified the normal way.
// Any other (self-signed) is trusted on first use: the first time, its fingerprint is
// saved to knownFile and announced on notice; afterwards only that fingerprint passes.
func ClientConfig(server, knownFile string, notice io.Writer) *tls.Config {
	return clientConfig(server, knownFile, notice, nil)
}

// clientConfig is ClientConfig with the trusted authorities as a parameter: nil means
// the system's (what real use wants); the tests pass their own test authority.
func clientConfig(server, knownFile string, notice io.Writer, roots *x509.CertPool) *tls.Config {
	host, _, err := net.SplitHostPort(server)
	if err != nil {
		host = server
	}
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		// skipping the standard checks is what lets a self-signed certificate work;
		// VerifyPeerCertificate below does the checking instead (authority first, then
		// fingerprint), so the connection is never left unverified
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
			certs := make([]*x509.Certificate, 0, len(raw))
			for _, der := range raw {
				cert, err := x509.ParseCertificate(der)
				if err != nil {
					return err
				}
				certs = append(certs, cert)
			}
			if len(certs) == 0 {
				return errors.New("the server sent no certificate")
			}
			// a certificate from a trusted authority (Let's Encrypt, a company CA installed
			// on this machine) is checked the way a browser checks it, with no pinning, so
			// renewals with a new key keep working
			verifyErr := verifyChain(certs, host, roots)
			if verifyErr == nil {
				return nil
			}
			// a certificate that claims an authority but didn't verify will be pinned below,
			// and refused after the server renews it with a new key: say so on first use
			warning := ""
			if !selfSigned(certs[0]) {
				warning = fmt.Sprintf("* warning: this certificate is issued by %q but didn't verify for %s (%v). "+
					"Connecting by IP instead of the domain name? Or the server uses cert.pem instead of fullchain.pem? "+
					"It's trusted on first use for now, and will be refused after the server renews it\n",
					certs[0].Issuer.CommonName, host, verifyErr)
			}
			// otherwise (self-signed, or not verifiable): trust on first use
			return checkKnown(server, Fingerprint(certs[0]), knownFile, notice, warning)
		},
	}
}

// verifyChain checks the chain the normal way, for host (empty host = don't check the
// name), under roots (nil = the system's trusted authorities). nil means valid.
func verifyChain(certs []*x509.Certificate, host string, roots *x509.CertPool) error {
	intermediates := x509.NewCertPool()
	for _, c := range certs[1:] {
		intermediates.AddCert(c)
	}
	_, err := certs[0].Verify(x509.VerifyOptions{DNSName: host, Roots: roots, Intermediates: intermediates})
	return err
}

// selfSigned reports whether the certificate signed itself, like teachat's own. It checks
// the signature directly with the certificate's own key: CheckSignatureFrom would refuse,
// because it requires the signer to be marked as an authority, and ours isn't.
func selfSigned(cert *x509.Certificate) bool {
	return bytes.Equal(cert.RawIssuer, cert.RawSubject) &&
		cert.CheckSignature(cert.SignatureAlgorithm, cert.RawTBSCertificate, cert.Signature) == nil
}

// checkKnown is trust on first use: unknown server → remember it; known server with
// the same fingerprint → fine; known server with a different one → refuse.
func checkKnown(server, fp, knownFile string, notice io.Writer, warning string) error {
	known, err := lookup(server, knownFile)
	if err != nil {
		return err
	}
	switch {
	case known == fp:
		return nil
	case known != "":
		return fmt.Errorf(`the identity of server %s has changed
  expected %s
  got      %s
Someone may be intercepting the connection, or the server's certificate was regenerated.
If you trust the change, delete the line for %s in %s and connect again`,
			server, known, fp, server, tilde(knownFile))
	}
	f, err := os.OpenFile(knownFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "%s %s\n", server, fp); err != nil {
		return err
	}
	fmt.Fprintf(notice, "* trusting new server %s (fingerprint %s), saved in %s\n", server, fp, tilde(knownFile))
	fmt.Fprint(notice, warning)
	return nil
}

// tilde shortens a path under the home directory to "~/...", for messages
func tilde(path string) string {
	if home, err := os.UserHomeDir(); err == nil {
		if rest, ok := strings.CutPrefix(path, home+string(filepath.Separator)); ok {
			return "~/" + rest
		}
	}
	return path
}

// lookup finds server's fingerprint in knownFile, one "server fingerprint" pair per line.
// A missing file just means no server is known yet.
func lookup(server, knownFile string) (string, error) {
	f, err := os.Open(knownFile)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if name, fp, ok := strings.Cut(strings.TrimSpace(sc.Text()), " "); ok && name == server {
			return fp, nil
		}
	}
	return "", sc.Err()
}

// DialHint adds a hint to the most common mistake: one side using TLS and the other
// not. tlsOn says whether this client tried TLS.
func DialHint(err error, tlsOn bool) error {
	msg := err.Error()
	switch {
	case tlsOn && strings.Contains(msg, "first record does not look like a TLS handshake"):
		return fmt.Errorf("%w (the server seems to run without TLS: try --no-tls)", err)
	case !tlsOn && strings.Contains(msg, "bad handshake"):
		return fmt.Errorf("%w (the server seems to require TLS: drop --no-tls)", err)
	}
	return err
}

// Server is what the server's main calls. It returns the TLS config and a one-line
// description for the startup message. With certFile and keyFile empty, it uses (and on
// the first run creates) the self-signed certificate in teachat's config directory.
// With both set, it serves the administrator's certificate instead, and writes a
// warning to warn if that certificate's chain looks broken.
func Server(certFile, keyFile string, warn io.Writer) (*tls.Config, string, error) {
	if (certFile == "") != (keyFile == "") {
		return nil, "", errors.New("--tls-cert and --tls-key must be used together")
	}
	if certFile != "" {
		return ProvidedConfig(certFile, keyFile, warn)
	}
	dir, err := Dir()
	if err != nil {
		return nil, "", err
	}
	cfg, fp, err := ServerConfig(dir)
	return cfg, "self-signed · fingerprint " + fp, err
}

// ProvidedConfig serves a certificate the administrator brings (Let's Encrypt's
// fullchain.pem + privkey.pem, or one from the company's CA). The files are checked on
// every new connection and reloaded when they change, so an automatic renewal (certbot)
// needs no restart. If the chain doesn't verify against this machine's trusted
// authorities, it writes a warning to warn: clients would fall back to first-use trust
// and refuse the server after its next renewal.
func ProvidedConfig(certFile, keyFile string, warn io.Writer) (*tls.Config, string, error) {
	return providedConfig(certFile, keyFile, warn, nil)
}

// providedConfig is ProvidedConfig with the trusted authorities as a parameter
// (nil = the system's); the tests pass their own.
func providedConfig(certFile, keyFile string, warn io.Writer, roots *x509.CertPool) (*tls.Config, string, error) {
	r := &reloader{certFile: certFile, keyFile: keyFile}
	cert, err := r.get(nil)
	if err != nil {
		return nil, "", fmt.Errorf("loading --tls-cert/--tls-key: %w", err)
	}
	leaf := cert.Leaf
	names := strings.Join(leaf.DNSNames, ", ")
	if names == "" {
		names = leaf.Subject.CommonName
	}
	desc := fmt.Sprintf("certificate for %s, issued by %s, valid until %s",
		names, leaf.Issuer.CommonName, leaf.NotAfter.Format("2006-01-02"))

	// check the chain like a client would (minus the hostname, which depends on how
	// clients connect). It's a warning, not an error: a private company CA installed on
	// every client is legitimate and would fail this check on the server.
	chain := []*x509.Certificate{leaf}
	for _, der := range cert.Certificate[1:] {
		if c, err := x509.ParseCertificate(der); err == nil {
			chain = append(chain, c)
		}
	}
	if err := verifyChain(chain, "", roots); err != nil {
		hint := ""
		if len(chain) == 1 && !selfSigned(leaf) {
			hint = "\n  The file has no intermediate certificate: with Let's Encrypt, use fullchain.pem, not cert.pem."
		}
		fmt.Fprintf(warn, "warning: the certificate doesn't verify against this machine's trusted authorities (%v).%s"+
			"\n  Clients that can't verify it fall back to first-use trust and will refuse the server after its next renewal."+
			"\n  A private company CA is fine if every client machine has it installed.\n", err, hint)
	}
	return &tls.Config{GetCertificate: r.get, MinVersion: tls.VersionTLS13}, desc, nil
}

// reloader keeps the administrator's certificate and reloads it when the files change
type reloader struct {
	certFile, keyFile string

	mu      sync.Mutex
	cert    *tls.Certificate
	modTime time.Time
}

// get returns the current certificate, reloading it if either file is newer than the
// loaded one. If a reload fails (say, a renewal wrote the certificate but not yet the
// key), it keeps serving the previous certificate instead of breaking connections.
func (r *reloader) get(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	mod, err := newest(r.certFile, r.keyFile)
	if err == nil && (r.cert == nil || mod.After(r.modTime)) {
		var cert tls.Certificate
		if cert, err = tls.LoadX509KeyPair(r.certFile, r.keyFile); err == nil {
			r.cert, r.modTime = &cert, mod
		}
	}
	if r.cert == nil {
		return nil, err
	}
	return r.cert, nil
}

// newest returns the latest modification time among the files
func newest(files ...string) (time.Time, error) {
	var latest time.Time
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			return time.Time{}, err
		}
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
	}
	return latest, nil
}

// Client is ClientConfig with teachat's default known_servers file: what the client's
// main calls.
func Client(server string, notice io.Writer) (*tls.Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	return ClientConfig(server, filepath.Join(dir, "known_servers"), notice), nil
}
