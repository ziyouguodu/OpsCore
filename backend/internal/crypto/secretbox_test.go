package crypto

import "testing"

func TestEncryptSecretDoesNotReturnPlaintextAndDecrypts(t *testing.T) {
	box, err := NewSecretBox("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("new secret box: %v", err)
	}

	ciphertext, err := box.Encrypt("root-password")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if ciphertext == "root-password" {
		t.Fatalf("expected encrypted secret not to equal plaintext")
	}

	plaintext, err := box.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if plaintext != "root-password" {
		t.Fatalf("unexpected plaintext: %q", plaintext)
	}
}

func TestSecretBoxRejectsShortKey(t *testing.T) {
	if _, err := NewSecretBox("short"); err == nil {
		t.Fatalf("expected short encryption key to be rejected")
	}
}

func TestIsEncryptedSecretDetectsVersionedCiphertext(t *testing.T) {
	if !IsEncryptedSecret("enc:v1:abc") {
		t.Fatalf("expected versioned ciphertext to be detected")
	}
	if IsEncryptedSecret("plain-password") {
		t.Fatalf("expected plaintext not to be detected as encrypted")
	}
}
