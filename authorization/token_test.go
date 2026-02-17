package authorization

import (
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestTokenEncryption(t *testing.T) {
	appID := "test-app-id"
	payload := &tokenBodyResponse{
		User:         "test-user",
		Roles:        []string{"admin", "user"},
		Capabilities: []string{"read", "write"},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	// Use the internal encrypt function from token.go
	cipherBytes, err := encrypt(b, appID)
	if err != nil {
		t.Fatalf("failed to encrypt payload: %v", err)
	}

	cipherHex := hex.EncodeToString(cipherBytes)
	t.Logf("Encrypted hex: %s", cipherHex)

	// Now decrypt it
	decryptedBytes, err := decrypt(cipherHex, appID)
	if err != nil {
		t.Fatalf("failed to decrypt payload: %v", err)
	}

	if string(decryptedBytes) != string(b) {
		t.Errorf("decrypted payload does not match original: got %s, want %s", string(decryptedBytes), string(b))
	}

	var decryptedPayload tokenBodyResponse
	if err := json.Unmarshal(decryptedBytes, &decryptedPayload); err != nil {
		t.Fatalf("failed to unmarshal decrypted payload: %v", err)
	}

	if decryptedPayload.User != payload.User {
		t.Errorf("decrypted user does not match: got %s, want %s", decryptedPayload.User, payload.User)
	}
}
