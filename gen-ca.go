package cproxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	rootPub = `
-----BEGIN CERTIFICATE-----
MIIDLDCCAhQCCQDOSy7YPYj0xjANBgkqhkiG9w0BAQsFADBYMQswCQYDVQQGEwJD
TjENMAsGA1UECAwERlJFRTENMAsGA1UEBwwERlJFRTENMAsGA1UECgwERlJFRTEN
MAsGA1UECwwERlJFRTENMAsGA1UEAwwERlJFRTAeFw0yMDAxMTMxNTU1MTlaFw0y
MTAxMTIxNTU1MTlaMFgxCzAJBgNVBAYTAkNOMQ0wCwYDVQQIDARGUkVFMQ0wCwYD
VQQHDARGUkVFMQ0wCwYDVQQKDARGUkVFMQ0wCwYDVQQLDARGUkVFMQ0wCwYDVQQD
DARGUkVFMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAyPKzycwVAzIM
DP1jMPmohbNC6yaXn931nHKFmJXCx5isVA126E7OnvKjnmbOlrn+zxRhLJBwFA+a
7yHmIdAkmSAkuJ3I66RaRDe7e18mCj2yTkBXOm10yLwaCpXDLpJp0WGX2oaHDKIS
nr35AnNNmLgEUM779rG0KyWFly/hXKSCCpgKFHpZHSRR5MLkwACBxC/fm9YEOuSP
dAWoxYXzJxezsA83RMWE487e5ywmtJmZPOtaxX5cMwuCUptfnXpjsDWt/Uc3JfaI
EQxKrbLHKIyLPkL8a4fsiQOoqGTTEo/aX8o4IC71URQ7dVbk/UZfqB5vITCMT3bo
Ys+7rGnHmQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQC3X36tfJoDdvHJF7xSkBTy
swNok7BWmNpve6X6bEyCHjIAgn8DBhAx3kQTppAFdxzTFp01QXO/G9X/n7SBHo21
F8dPf4sJLioEWwmVgw1ivaQbXettiaIDeVcxziifJi/OctJP7j5GsPDf4d7OzIgA
fMrtEGRAu9BNdifSJ8Dnw5zSCDHgtFcu0KCnXxBznJR7Mo2NLOUFR/3IIrMFKQPi
OZKhM+MgG+57UhPNTLQhpUkx7h+XtmuDSg/5mHjmz1bqNOvHWDacBBT50NwnlK9j
DtVsDKLaCv0/Cp+V/DbBp3jfEqSbX7k6TBUnVtcAcGscTx3ZeZ3aFMuqgeR+NeUn
-----END CERTIFICATE-----
`
	rootPriv = `
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAyPKzycwVAzIMDP1jMPmohbNC6yaXn931nHKFmJXCx5isVA12
6E7OnvKjnmbOlrn+zxRhLJBwFA+a7yHmIdAkmSAkuJ3I66RaRDe7e18mCj2yTkBX
Om10yLwaCpXDLpJp0WGX2oaHDKISnr35AnNNmLgEUM779rG0KyWFly/hXKSCCpgK
FHpZHSRR5MLkwACBxC/fm9YEOuSPdAWoxYXzJxezsA83RMWE487e5ywmtJmZPOta
xX5cMwuCUptfnXpjsDWt/Uc3JfaIEQxKrbLHKIyLPkL8a4fsiQOoqGTTEo/aX8o4
IC71URQ7dVbk/UZfqB5vITCMT3boYs+7rGnHmQIDAQABAoIBADfmRBtT2ViNOIr4
hfpeyQGAb5Iopy9CuItv1DgxGQEbOH0dTcGsApB24Qs0gC2vyfFjMvEJsRPzj18M
aA9p7nRmW7C7u+PJUY7+jfnw6w0YQpzAC0PmpQEeSoQ9SxGOiz9CzdJtb+4Uu+dK
45VJ7AEa16B/I9ppbrw98N6w5Bk38AASM/HPxqIaRrXUhR+e5BBG2p7n7kNpd6gM
isAmfBgrTgeDPxUnwTqIQ/K/DFMvzm4orqhLPqnXhWhlFUazX9pCmiOrQ81kiTbT
cbqv5xH7GE+YxndgkNh7A8U7IAW7o6MHdbgAMrilVFA3OOL8BLJ7YkoaAnVWJGI2
o8xPkrECgYEA4/X6PJ76klMiIYA/h5xF113xO004dDwASTQZo07ZroZO2BUzLiwd
bKpYKSYre9v5pE24O4kemTFDfgRJY7YeX2N3FinLS4K6zbu7496G5nkxWneWaV6Z
8ePR5IPIlAOIy+9wX5YboZi3PwRuLgySX9f2xEUpxZyN3bJdYlSaGy0CgYEA4aol
oXqrps2XLW2jIKUDiLlqrdS5GLQ0u/ToHteA3Fteiv4FNmBNIAOFebXIeGA6C5IS
S1svAiqu4Mt74D53i6C5MgDusXr78MTh96ab805r/qaArZ2pA/8irnkEs4oKVkhQ
2TT3zv9J6uixp4iDP/bMJQ9xssEcxsbteB+xsZ0CgYEArKl38wibU89h76v68pU8
FScTe04+71MvCENNE/O6T0VtXJ+aF2PUmaTgh7Jghz0Tdg5j97whD/lPXJiUmdCs
aqWk4oWfdL89DG0goDTBSroK1rHznDXKNnvPU905RFr09zqRi+TfYuOQEEwjw/9D
sxKZ1wln3UR586yQrNTVsLUCgYEAgE7tDI6iMLpuvb675ODOTJwYYvQzti8oWMJc
hMTFmQU+kUrzjcJdt9kouFY6wO79sfyA+GXFKbc5LcmlCpCaCkL9acgL78/clj5r
uRL7UvEBCI6FVbHyGrqjbo6StL7FN9/wUEAEsqaG0dEyye4dqm3aDyxj2l5gzUjo
Vse2kiUCgYEAiW+12DpFV6y90D1i3xqxm4nQLT0Z9EB+ZC2wEuSgQpfl9pN2hWy9
+yQeP6sWx4viAS3RaK4F83mjH/0tLkThp+qJ/k/S9/lGwo9jl9GkYt5indbxGqjb
fOGk7h+1H+b8P82ow4rbH1N0B8uioE42cr084JWfwnIQUUQlXptRe7k=
-----END RSA PRIVATE KEY-----
`
	caPubPath = "crts/root.crt"
	caPrivPath = "crts/root.pem"
)

func getTopDomain(host string) (topDomain, domain string) {
	s := strings.Split(host, ":")
	domain = s[0]
	topDomain = domain
	sps := strings.Split(s[0], ".")
	if len(sps) > 2 {
		l := len(sps)
		topDomain = fmt.Sprintf("%s.%s", sps[l-2], sps[l-1])
	}
	return
}

func GetCAPairPath(domain string) (string, string) {

	caFile := fmt.Sprintf("crts/%s.crt", domain)
	keyFile := fmt.Sprintf("crts/%s.key", domain)
	if _, err := os.Stat(caFile); os.IsNotExist(err) {
		Sigin(domain)
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		Sigin(domain)
	}
	return caFile, keyFile
}

func init() {

	if _, err := os.Stat(caPubPath); os.IsNotExist(err) {
		WriteToFile(caPubPath, []byte(rootPub))
	}
	if _, err := os.Stat(caPrivPath); os.IsNotExist(err) {
		WriteToFile(caPrivPath, []byte(rootPriv))
	}
}

func Sigin(host string) {

	caPublicKeyFile, err := ioutil.ReadFile(caPubPath)
	if err != nil {
		panic(err)
	}
	pemBlock, _ := pem.Decode(caPublicKeyFile)
	if pemBlock == nil {
		panic("pem.Decode failed")
	}
	caCRT, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		panic(err)
	}
	//tls.LoadX509KeyPair(certFile, keyFile)
	//      private key
	caPrivateKeyFile, err := ioutil.ReadFile(caPrivPath)
	if err != nil {
		panic(err)
	}
	pemBlock, _ = pem.Decode(caPrivateKeyFile)
	if pemBlock == nil {
		panic("pem.Decode failed")
	}
	//der, err := x509.DecryptPEMBlock(pemBlock, []byte("password"))
	//if err != nil {
	//	panic(err)
	//}
	caPrivateKey, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		panic(err)
	}

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	kf := fmt.Sprintf("crts/%s.key", host)
	keyToFile(kf, leafKey)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 365)

	leafTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"FREE"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	if ip := net.ParseIP(host); ip != nil {
		leafTemplate.IPAddresses = append(leafTemplate.IPAddresses, ip)
	} else {
		leafTemplate.Subject.CommonName = host
		leafTemplate.DNSNames = append(leafTemplate.DNSNames, host)

	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &leafTemplate, caCRT, &leafKey.PublicKey, caPrivateKey)
	if err != nil {
		panic(err)
	}
	//debugCertToFile(fmt.Sprintf("%s.debug.crt", host), derBytes)
	certToFile(fmt.Sprintf("crts/%s.crt", host), derBytes)
}

// keyToFile writes a PEM serialization of |key| to a new file called
// |filename|.
func keyToFile(filename string, key *rsa.PrivateKey) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	b := x509.MarshalPKCS1PrivateKey(key)
	// if err != nil {
	//     fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
	//     os.Exit(2)
	// }
	if err := pem.Encode(file, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}); err != nil {
		panic(err)
	}
}

func certToFile(filename string, derBytes []byte) {
	certOut, err := os.Create(filename)
	if err != nil {
		log.Fatalf("failed to open cert.pem for writing: %s", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("failed to write data to cert.pem: %s", err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("error closing cert.pem: %s", err)
	}
}

// debugCertToFile writes a PEM serialization and OpenSSL debugging dump of
// |derBytes| to a new file called |filename|.
func debugCertToFile(filename string, derBytes []byte) {
	cmd := exec.Command("openssl", "x509", "-text", "-inform", "DER")

	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	cmd.Stdout = file
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}
	if _, err := stdin.Write(derBytes); err != nil {
		panic(err)
	}
	stdin.Close()
	if err := cmd.Wait(); err != nil {
		panic(err)
	}
}
