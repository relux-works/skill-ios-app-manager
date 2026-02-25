package push

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateJWTStructureAndClaims(t *testing.T) {
	t.Parallel()

	keyPath, key := writeTestP8Key(t)
	fixedTime := time.Unix(1_700_000_000, 0)

	sender := NewSender(SenderConfig{
		KeyPath:  keyPath,
		KeyID:    "ABC123DEF4",
		TeamID:   "ABCDE12345",
		BundleID: "com.example.demo",
		Clock: func() time.Time {
			return fixedTime
		},
	})

	token, err := sender.GenerateJWT()
	if err != nil {
		t.Fatalf("GenerateJWT() error = %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT segments = %d, want 3; token=%q", len(parts), token)
	}

	headerSegment, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("DecodeString(header) error = %v", err)
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerSegment, &header); err != nil {
		t.Fatalf("Unmarshal(header) error = %v", err)
	}
	if header.Alg != "ES256" || header.Kid != "ABC123DEF4" {
		t.Fatalf("header = %#v, want alg=ES256, kid=ABC123DEF4", header)
	}

	claimsSegment, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("DecodeString(claims) error = %v", err)
	}
	var claims struct {
		Iss string `json:"iss"`
		Iat int64  `json:"iat"`
	}
	if err := json.Unmarshal(claimsSegment, &claims); err != nil {
		t.Fatalf("Unmarshal(claims) error = %v", err)
	}
	if claims.Iss != "ABCDE12345" || claims.Iat != fixedTime.Unix() {
		t.Fatalf("claims = %#v, want iss=ABCDE12345 and iat=%d", claims, fixedTime.Unix())
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("DecodeString(signature) error = %v", err)
	}
	if len(signature) != 64 {
		t.Fatalf("signature length = %d, want 64", len(signature))
	}

	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(big.Int).SetBytes(signature[:32])
	sInt := new(big.Int).SetBytes(signature[32:])
	if !ecdsa.Verify(&key.PublicKey, digest[:], r, sInt) {
		t.Fatal("JWT signature verification failed")
	}
}

func TestBuildPayloadDefault(t *testing.T) {
	t.Parallel()

	payload, err := BuildPayload("DemoApp", nil)
	if err != nil {
		t.Fatalf("BuildPayload(default) error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal(default payload) error = %v", err)
	}

	aps, ok := decoded["aps"].(map[string]any)
	if !ok {
		t.Fatalf("default payload aps = %#v, want object", decoded["aps"])
	}
	if aps["alert"] != "Test push from DemoApp" {
		t.Fatalf("default payload alert = %#v, want %q", aps["alert"], "Test push from DemoApp")
	}
	if aps["sound"] != "default" {
		t.Fatalf("default payload sound = %#v, want %q", aps["sound"], "default")
	}
}

func TestBuildPayloadCustom(t *testing.T) {
	t.Parallel()

	custom := []byte(`{"aps":{"alert":"Hello"},"meta":{"id":7}}`)

	payload, err := BuildPayload("DemoApp", custom)
	if err != nil {
		t.Fatalf("BuildPayload(custom) error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal(custom payload) error = %v", err)
	}

	if _, ok := decoded["meta"].(map[string]any); !ok {
		t.Fatalf("custom payload meta = %#v, want object", decoded["meta"])
	}
}

func TestBuildPayloadRejectsNonObjectJSON(t *testing.T) {
	t.Parallel()

	if _, err := BuildPayload("DemoApp", []byte(`["a","b"]`)); err == nil {
		t.Fatal("BuildPayload(non-object) error = nil, want validation error")
	}
}

func TestSendUsesAPNSHeadersAndParsesResponse(t *testing.T) {
	t.Parallel()

	keyPath, _ := writeTestP8Key(t)
	fixedTime := time.Unix(1_700_000_123, 0)

	var capturedURL string
	var capturedAuthHeader string
	var capturedTopic string
	var capturedPushType string
	var capturedBody []byte

	mockClient := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			capturedURL = request.URL.String()
			capturedAuthHeader = request.Header.Get("authorization")
			capturedTopic = request.Header.Get("apns-topic")
			capturedPushType = request.Header.Get("apns-push-type")

			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatalf("ReadAll(request.Body) error = %v", err)
			}
			capturedBody = body

			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Header: http.Header{
					"Apns-Id": []string{"apns-id-123"},
				},
				Body:    io.NopCloser(strings.NewReader(`{"reason":"BadDeviceToken"}`)),
				Request: request,
			}, nil
		}),
	}

	sender := NewSender(SenderConfig{
		KeyPath:    keyPath,
		KeyID:      "ABC123DEF4",
		TeamID:     "ABCDE12345",
		BundleID:   "com.example.demo",
		HTTPClient: mockClient,
		Clock: func() time.Time {
			return fixedTime
		},
	})

	response, err := sender.Send(context.Background(), SendRequest{
		DeviceToken: "0011223344556677",
		Environment: EnvironmentDevelopment,
		AppName:     "DemoApp",
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if capturedURL != developmentEndpoint+"/3/device/0011223344556677" {
		t.Fatalf("request URL = %q, want %q", capturedURL, developmentEndpoint+"/3/device/0011223344556677")
	}
	if capturedTopic != "com.example.demo" {
		t.Fatalf("apns-topic = %q, want %q", capturedTopic, "com.example.demo")
	}
	if capturedPushType != "alert" {
		t.Fatalf("apns-push-type = %q, want %q", capturedPushType, "alert")
	}
	if !strings.HasPrefix(strings.ToLower(capturedAuthHeader), "bearer ") {
		t.Fatalf("authorization header = %q, want bearer token", capturedAuthHeader)
	}
	if tokenParts := strings.Split(strings.TrimSpace(strings.TrimPrefix(capturedAuthHeader, "bearer ")), "."); len(tokenParts) != 3 {
		t.Fatalf("authorization bearer token segments = %d, want 3", len(tokenParts))
	}

	var payload map[string]any
	if err := json.Unmarshal(capturedBody, &payload); err != nil {
		t.Fatalf("Unmarshal(captured body) error = %v", err)
	}
	aps, ok := payload["aps"].(map[string]any)
	if !ok {
		t.Fatalf("captured payload aps = %#v, want object", payload["aps"])
	}
	if aps["alert"] != "Test push from DemoApp" {
		t.Fatalf("captured payload alert = %#v, want %q", aps["alert"], "Test push from DemoApp")
	}

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("response.StatusCode = %d, want %d", response.StatusCode, http.StatusBadRequest)
	}
	if response.APNSID != "apns-id-123" {
		t.Fatalf("response.APNSID = %q, want %q", response.APNSID, "apns-id-123")
	}
	if response.ErrorDescription != "BadDeviceToken" {
		t.Fatalf("response.ErrorDescription = %q, want %q", response.ErrorDescription, "BadDeviceToken")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func writeTestP8Key(t *testing.T) (string, *ecdsa.PrivateKey) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey() error = %v", err)
	}

	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("x509.MarshalPKCS8PrivateKey() error = %v", err)
	}

	path := filepath.Join(t.TempDir(), "AuthKey_TEST.p8")
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, pemData, 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}

	return path, key
}
