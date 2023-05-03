package gokey

import (
	"bytes"
	"crypto"
	"encoding/asn1"
	"encoding/hex"
	"encoding/pem"
	"os"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/crypto/ed25519"
)

var passSpec = &PasswordSpec{16, 3, 3, 2, 1, ""}

func TestGetPass(t *testing.T) {
	pass1Seed1, err := GenerateEncryptedKeySeed("pass1")
	if err != nil {
		t.Fatal(err)
	}

	pass1Seed2, err := GenerateEncryptedKeySeed("pass1")
	if err != nil {
		t.Fatal(err)
	}

	pass1Example1, err := GetPass("pass1", "example.com", nil, passSpec)
	if err != nil {
		t.Fatal(err)
	}

	pass1Example2, err := GetPass("pass1", "example2.com", nil, passSpec)
	if err != nil {
		t.Fatal(err)
	}

	pass2Example1, err := GetPass("pass2", "example.com", nil, passSpec)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Compare(pass1Example1, pass1Example2) == 0 {
		t.Fatal("passwords match for different realms")
	}

	if strings.Compare(pass1Example1, pass2Example1) == 0 {
		t.Fatal("passwords match for different master passwords")
	}

	pass1Example1Seed1, err := GetPass("pass1", "example.com", pass1Seed1, passSpec)
	if err != nil {
		t.Fatal(err)
	}

	pass1Example1Seed2, err := GetPass("pass1", "example.com", pass1Seed2, passSpec)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Compare(pass1Example1, pass1Example1Seed1) == 0 {
		t.Fatal("passwords match for seeded and non-seeded master password")
	}

	if strings.Compare(pass1Example1Seed1, pass1Example1Seed2) == 0 {
		t.Fatal("passwords match for different seeds")
	}

	pass1Example1Retry, err := GetPass("pass1", "example.com", nil, passSpec)
	if err != nil {
		t.Fatal(err)
	}

	pass1Example1Seed1Retry, err := GetPass("pass1", "example.com", pass1Seed1, passSpec)
	if err != nil {
		t.Fatal(err)
	}

	if (strings.Compare(pass1Example1, pass1Example1Retry) != 0) || (strings.Compare(pass1Example1Seed1, pass1Example1Seed1Retry) != 0) {
		t.Fatal("passwords with same invocation options do not match")
	}

	// Testing GOKEY_MASTER environment variable
	os.Setenv("GOKEY_MASTER", "pass1")
	passWithEnv, err := GetPass(os.Getenv("GOKEY_MASTER"), "example.com", nil, passSpec)
	if err != nil {
		t.Fatal(err)
	}
	passWithoutEnv, err := GetPass("pass1", "example.com", nil, passSpec)
	if err != nil {
		t.Fatal(err)
	}
	os.Unsetenv("GOKEY_MASTER")
	if strings.Compare(passWithEnv, passWithoutEnv) != 0 {
		t.Fatal("pasword with env GOKEY_MASTER set does not match user supplied password")
	}

}

func keyToBytes(key crypto.PrivateKey, t *testing.T) []byte {
	buf := bytes.NewBuffer(nil)

	err := EncodeToPem(key, buf)
	if err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func testGetKeyType(kt KeyType, t *testing.T) {
	pass1Seed1, err := GenerateEncryptedKeySeed("pass1")
	if err != nil {
		t.Fatal(err)
	}

	pass1Seed2, err := GenerateEncryptedKeySeed("pass1")
	if err != nil {
		t.Fatal(err)
	}

	key1Example1, err := GetKey("pass1", "example.com", nil, kt, true)
	if err != nil {
		t.Fatal(err)
	}

	key1Example2, err := GetKey("pass1", "example2.com", nil, kt, true)
	if err != nil {
		t.Fatal(err)
	}

	key2Example1, err := GetKey("pass2", "example.com", nil, kt, true)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(keyToBytes(key1Example1, t), keyToBytes(key1Example2, t)) == 0 {
		t.Fatal("keys match for different realms")
	}

	if bytes.Compare(keyToBytes(key1Example1, t), keyToBytes(key2Example1, t)) == 0 {
		t.Fatal("keys match for different master passwords")
	}

	key1Example1Seed1, err := GetKey("pass1", "example.com", pass1Seed1, kt, false)
	if err != nil {
		t.Fatal(err)
	}

	key1Example1Seed2, err := GetKey("pass1", "example.com", pass1Seed2, kt, false)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(keyToBytes(key1Example1, t), keyToBytes(key1Example1Seed1, t)) == 0 {
		t.Fatal("keys match for seeded and non-seeded master password")
	}

	if bytes.Compare(keyToBytes(key1Example1Seed1, t), keyToBytes(key1Example1Seed2, t)) == 0 {
		t.Fatal("keys match for different seeds")
	}

	key1Example1Retry, err := GetKey("pass1", "example.com", nil, kt, true)
	if err != nil {
		t.Fatal(err)
	}

	key1Example1Seed1Retry, err := GetKey("pass1", "example.com", pass1Seed1, kt, false)
	if err != nil {
		t.Fatal(err)
	}

	if (bytes.Compare(keyToBytes(key1Example1, t), keyToBytes(key1Example1Retry, t)) != 0) || (bytes.Compare(keyToBytes(key1Example1Seed1, t), keyToBytes(key1Example1Seed1Retry, t)) != 0) {
		t.Fatal("keys with same invocation options do not match")
	}
}

func TestGetKey(t *testing.T) {
	for _, kt := range []KeyType{
		EC256,
		EC384,
		EC521,
		RSA2048,
		RSA4096,
		X25519,
		ED25519,
	} {
		t.Run(kt.String(), func(t *testing.T) {
			testGetKeyType(kt, t)
		})
	}
}

func TestGetKeyUnsafe(t *testing.T) {
	_, err := GetKey("pass1", "example.com", nil, EC256, false)
	if err == nil {
		t.Fatal("allowed unsafe key generation")
	}
}

func parse25519(t *testing.T, keyType KeyType, refKey string, refKeyBytes []byte) {
	var suffix int

	switch keyType {
	case X25519:
		suffix = x25519OidSuffix
	case ED25519:
		suffix = ed25519OidSuffix
	}

	block, _ := pem.Decode([]byte(refKey))
	if block == nil {
		t.Fatal("unable to pem-decode x25519 key")
	}

	var a25519 asn25519
	_, err := asn1.Unmarshal(block.Bytes, &a25519)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(a25519.AlgId.Algorithm, asn1.ObjectIdentifier{1, 3, 101, suffix}) || !reflect.DeepEqual(a25519.PrivateKey, refKeyBytes) {
		t.Fatalf("invalid %v key after parsing", keyType)
	}
}

func TestParseX25519(t *testing.T) {
	// generated by
	// $ openssl genpkey -algorithm x25519
	x25519Openssl := `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VuBCIEIEBQhR7E5x8vlFgWOhonI+/H3DY1R9mCh6wdwd8Hkgl1
-----END PRIVATE KEY-----`

	// if you parse the key above the actual key bytes should be the following
	// $ echo <above> | openssl asn1parse -i
	x25519OpensslKeyBytes, err := hex.DecodeString("04204050851EC4E71F2F9458163A1A2723EFC7DC363547D98287AC1DC1DF07920975")
	if err != nil {
		t.Fatal(err)
	}

	parse25519(t, X25519, x25519Openssl, x25519OpensslKeyBytes)
}

func TestParseEd25519(t *testing.T) {
	// taken from https://github.com/openssl/openssl/blob/60bbed3ff6716e8f1358396acc772908a758a0a0/test/certs/client-ed25519-key.pem
	ed25519Openssl := `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEINZzpIpIiXXsKx4M7mUr2cb+DMfgHyu2msRAgNa5CxJJ
-----END PRIVATE KEY-----`

	// if you parse the key above the actual key bytes should be the following
	// $ echo <above> | openssl asn1parse -i
	ed25519OpensslKeyBytes, err := hex.DecodeString("0420D673A48A488975EC2B1E0CEE652BD9C6FE0CC7E01F2BB69AC44080D6B90B1249")
	if err != nil {
		t.Fatal(err)
	}

	parse25519(t, ED25519, ed25519Openssl, ed25519OpensslKeyBytes)
}

func gen25519(t *testing.T, keyType KeyType) {
	seed, err := GenerateEncryptedKeySeed("pass1")
	if err != nil {
		t.Fatal(err)
	}

	key, err := GetKey("pass1", "example.com", seed, keyType, false)
	if err != nil {
		t.Fatal(err)
	}

	var b strings.Builder
	err = EncodeToPem(key, &b)
	if err != nil {
		t.Fatal(err)
	}

	var keyBytes []byte
	switch keyType {
	case X25519:
		keyBytes = key.(x25519PrivateKey)[:]
	case ED25519:
		keyBytes = key.(*ed25519.PrivateKey).Seed()
	}

	parse25519(t, keyType, b.String(), append([]byte{0x04, 0x20}, keyBytes...))
}

func TestGenX25519(t *testing.T) {
	gen25519(t, X25519)
}

func TestGenEd25519(t *testing.T) {
	gen25519(t, ED25519)
}
