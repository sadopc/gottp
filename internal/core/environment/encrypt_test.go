package environment

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	passphrase := "test-passphrase-123"
	plaintext := "my-secret-api-key"

	encrypted, err := EncryptValue(plaintext, passphrase)
	if err != nil {
		t.Fatal(err)
	}

	if !IsEncrypted(encrypted) {
		t.Error("encrypted value should have prefix")
	}

	if encrypted == plaintext {
		t.Error("encrypted should differ from plaintext")
	}

	decrypted, err := DecryptValue(encrypted, passphrase)
	if err != nil {
		t.Fatal(err)
	}

	if decrypted != plaintext {
		t.Errorf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	encrypted, err := EncryptValue("secret", "correct-passphrase")
	if err != nil {
		t.Fatal(err)
	}

	_, err = DecryptValue(encrypted, "wrong-passphrase")
	if err == nil {
		t.Error("expected error with wrong passphrase")
	}
}

func TestDecryptPlaintext(t *testing.T) {
	// DecryptValue should return plain values as-is
	result, err := DecryptValue("not-encrypted-value", "any-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if result != "not-encrypted-value" {
		t.Errorf("expected plain value back, got %q", result)
	}
}

func TestIsEncrypted(t *testing.T) {
	if IsEncrypted("plain-value") {
		t.Error("plain value should not be detected as encrypted")
	}
	if !IsEncrypted("enc:v1:somedata") {
		t.Error("prefixed value should be detected as encrypted")
	}
}

func TestEncryptDifferentNonces(t *testing.T) {
	passphrase := "test"
	plaintext := "same-value"

	enc1, _ := EncryptValue(plaintext, passphrase)
	enc2, _ := EncryptValue(plaintext, passphrase)

	if enc1 == enc2 {
		t.Error("two encryptions of the same value should produce different ciphertexts")
	}

	// But both should decrypt to the same value
	dec1, _ := DecryptValue(enc1, passphrase)
	dec2, _ := DecryptValue(enc2, passphrase)
	if dec1 != dec2 {
		t.Error("both should decrypt to the same value")
	}
}

func TestEncryptEmptyString(t *testing.T) {
	encrypted, err := EncryptValue("", "passphrase")
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := DecryptValue(encrypted, "passphrase")
	if err != nil {
		t.Fatal(err)
	}

	if decrypted != "" {
		t.Errorf("expected empty string, got %q", decrypted)
	}
}
