package authorization

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	coreauth "github.com/GPA-Gruppo-Progetti-Avanzati-SRL/go-core-app/authorization"
	"github.com/danielgtaylor/huma/v2"
)

var TokenResponses = map[string]*huma.Response{
	"200": {Description: "OK", Content: map[string]*huma.MediaType{"text/plain": {Schema: &huma.Schema{Type: huma.TypeString}}}},
}

type RawStringOutput struct {
	ContentType string `header:"Content-Type"`
	Body        []byte
}

func (r *RawStringOutput) MarshalJSON() ([]byte, error) {
	return r.Body, nil
}

type tokenBodyResponse struct {
	User         string           `json:"user"`
	Roles        []string         `json:"roles"`
	Capabilities []string         `json:"capabilities"`
	Apps         []*coreauth.App  `json:"apps"`
	Paths        []*coreauth.Path `json:"paths"`
}

// whoamiRequest consente di passare opzionalmente l'AppId via header
type tokenRequest struct {
	AppID string `header:"AppId"  required:"true"`
}

var TokenOperation = huma.Operation{
	OperationID:   "Token",
	Method:        http.MethodGet,
	Path:          "/api/token",
	Summary:       "Informazioni sull'utente corrente e i permessi derivati dai ruoli",
	Tags:          []string{"system"},
	DefaultStatus: http.StatusOK,
	Responses:     TokenResponses,
}

// Token restituisce informazioni sull'utente e capacità derivate dai ruoli, leggendo dal context.
// Il risultato è cifrato in formato hex usando l'AppID come chiave.
func Token(ctx context.Context, i *tokenRequest) (*RawStringOutput, error) {
	// Estrae user e roles dal context impostato dal middleware di autorizzazione
	var user string
	if v := ctx.Value("user"); v != nil {
		if s, ok := v.(string); ok {
			user = s
		}
	}
	var roles []string
	if v := ctx.Value("roles"); v != nil {
		if rr, ok := v.([]string); ok {
			roles = rr
		}
	}

	// Recupera l'authorizer dal context, se presente
	var caps []string
	var apps []*coreauth.App
	var paths []*coreauth.Path
	if v := ctx.Value("authorizer"); v != nil {
		if auth, ok := v.(coreauth.Authorizer); ok && auth != nil {

			apps = auth.GetApps(roles)
			paths = auth.GetPaths(roles, i.AppID)
			caps = auth.GetCapabilities(roles, i.AppID)

		}
	}

	body := &tokenBodyResponse{User: user, Roles: roles, Capabilities: caps, Apps: apps, Paths: paths}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	cipherText, err := encrypt(b, i.AppID)
	if err != nil {
		return nil, huma.Error500InternalServerError("encryption error", err)
	}

	return &RawStringOutput{Body: []byte(hex.EncodeToString(cipherText)), ContentType: "text/plain"}, nil

}

func encrypt(plaintext []byte, keyStr string) ([]byte, error) {
	// Crea una chiave di 32 byte dall'AppID usando SHA-256
	hash := sha256.Sum256([]byte(keyStr))
	key := hash[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Concatena il nonce al ciphertext.
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// decrypt decritta un messaggio esadecimale usando l'AppID come chiave.
// Il nonce è atteso all'inizio del ciphertext decodificato da hex.
func decrypt(ciphertextHex string, keyStr string) ([]byte, error) {
	ciphertext, err := hex.DecodeString(ciphertextHex)
	if err != nil {
		return nil, err
	}

	hash := sha256.Sum256([]byte(keyStr))
	key := hash[:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
