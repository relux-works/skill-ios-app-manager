package push

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	developmentEndpoint = "https://api.development.push.apple.com"
	productionEndpoint  = "https://api.push.apple.com"
	maxResponseBodySize = 1 << 20 // 1 MiB safety limit
	defaultHTTPTimeout  = 15 * time.Second
)

// Environment defines APNs environment.
type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
)

// ParseEnvironment converts CLI value into APNs environment.
func ParseEnvironment(value string) (Environment, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "dev", "development":
		return EnvironmentDevelopment, nil
	case "prod", "production":
		return EnvironmentProduction, nil
	default:
		return "", fmt.Errorf("invalid push environment %q: expected dev|prod", value)
	}
}

func (e Environment) endpoint() string {
	if e == EnvironmentProduction {
		return productionEndpoint
	}
	return developmentEndpoint
}

// SenderConfig configures APNs sender.
type SenderConfig struct {
	KeyPath  string
	KeyID    string
	TeamID   string
	BundleID string

	HTTPClient *http.Client
	Clock      func() time.Time
}

// SendRequest is an APNs send request payload.
type SendRequest struct {
	DeviceToken string
	Environment Environment
	AppName     string
	Payload     []byte
}

// SendResponse captures APNs response metadata.
type SendResponse struct {
	StatusCode       int
	APNSID           string
	ErrorDescription string
}

// Sender sends pushes to APNs using token-based authentication.
type Sender struct {
	config SenderConfig
	client *http.Client
	clock  func() time.Time
}

// NewSender builds APNs sender with sane defaults.
func NewSender(cfg SenderConfig) *Sender {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{
			Transport: &http.Transport{ForceAttemptHTTP2: true},
			Timeout:   defaultHTTPTimeout,
		}
	}

	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}

	return &Sender{
		config: cfg,
		client: client,
		clock:  clock,
	}
}

// GenerateJWT creates APNs provider token signed with ES256.
func (s *Sender) GenerateJWT() (string, error) {
	if err := validateSenderConfig(s.config); err != nil {
		return "", err
	}

	key, err := loadPrivateKey(s.config.KeyPath)
	if err != nil {
		return "", err
	}

	header, err := json.Marshal(struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}{
		Alg: "ES256",
		Kid: strings.TrimSpace(s.config.KeyID),
	})
	if err != nil {
		return "", fmt.Errorf("marshal JWT header: %w", err)
	}

	claims, err := json.Marshal(struct {
		Iss string `json:"iss"`
		Iat int64  `json:"iat"`
	}{
		Iss: strings.TrimSpace(s.config.TeamID),
		Iat: s.clock().Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("marshal JWT claims: %w", err)
	}

	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	signature, err := signES256(signingInput, key)
	if err != nil {
		return "", err
	}

	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// Send sends a push payload to APNs and returns APNs response metadata.
func (s *Sender) Send(ctx context.Context, request SendRequest) (SendResponse, error) {
	if err := validateSenderConfig(s.config); err != nil {
		return SendResponse{}, err
	}

	deviceToken := strings.TrimSpace(request.DeviceToken)
	if deviceToken == "" {
		return SendResponse{}, errors.New("device token is required")
	}

	environment, err := ParseEnvironment(string(request.Environment))
	if err != nil {
		return SendResponse{}, err
	}

	payload, err := BuildPayload(request.AppName, request.Payload)
	if err != nil {
		return SendResponse{}, err
	}

	jwt, err := s.GenerateJWT()
	if err != nil {
		return SendResponse{}, fmt.Errorf("generate APNs JWT: %w", err)
	}

	endpoint := environment.endpoint() + "/3/device/" + deviceToken
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return SendResponse{}, fmt.Errorf("build APNs request: %w", err)
	}

	httpRequest.Header.Set("authorization", "bearer "+jwt)
	httpRequest.Header.Set("apns-topic", strings.TrimSpace(s.config.BundleID))
	httpRequest.Header.Set("apns-push-type", "alert")
	httpRequest.Header.Set("content-type", "application/json")

	httpResponse, err := s.client.Do(httpRequest)
	if err != nil {
		return SendResponse{}, fmt.Errorf("send APNs request: %w", err)
	}
	defer httpResponse.Body.Close()

	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxResponseBodySize))
	if err != nil {
		return SendResponse{}, fmt.Errorf("read APNs response body: %w", err)
	}

	response := SendResponse{
		StatusCode: httpResponse.StatusCode,
		APNSID:     strings.TrimSpace(httpResponse.Header.Get("apns-id")),
	}

	if trimmed := bytes.TrimSpace(body); len(trimmed) > 0 {
		var errorPayload struct {
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal(trimmed, &errorPayload); err == nil {
			response.ErrorDescription = strings.TrimSpace(errorPayload.Reason)
		} else if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
			response.ErrorDescription = string(trimmed)
		}
	}

	return response, nil
}

// BuildPayload returns custom payload if provided, otherwise default alert payload.
func BuildPayload(appName string, customPayload []byte) ([]byte, error) {
	if len(bytes.TrimSpace(customPayload)) == 0 {
		return DefaultPayload(appName)
	}

	var parsed any
	if err := json.Unmarshal(customPayload, &parsed); err != nil {
		return nil, fmt.Errorf("parse payload JSON: %w", err)
	}

	if _, ok := parsed.(map[string]any); !ok {
		return nil, errors.New("payload JSON must be an object")
	}

	normalized, err := json.Marshal(parsed)
	if err != nil {
		return nil, fmt.Errorf("normalize payload JSON: %w", err)
	}

	return normalized, nil
}

// DefaultPayload returns default alert payload for test push.
func DefaultPayload(appName string) ([]byte, error) {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "App"
	}

	return json.Marshal(map[string]any{
		"aps": map[string]any{
			"alert": fmt.Sprintf("Test push from %s", name),
			"sound": "default",
		},
	})
}

func validateSenderConfig(cfg SenderConfig) error {
	if strings.TrimSpace(cfg.KeyPath) == "" {
		return errors.New("push key path is required")
	}
	if strings.TrimSpace(cfg.KeyID) == "" {
		return errors.New("push key id is required")
	}
	if strings.TrimSpace(cfg.TeamID) == "" {
		return errors.New("team id is required")
	}
	if strings.TrimSpace(cfg.BundleID) == "" {
		return errors.New("bundle id is required")
	}

	return nil
}

func loadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read APNs key %q: %w", path, err)
	}

	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, errors.New("decode APNs key PEM: no PEM block found")
	}

	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse APNs key as PKCS8: %w", err)
	}

	key, ok := keyAny.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("APNs key must be ECDSA, got %T", keyAny)
	}

	if key.Curve.Params().Name != elliptic.P256().Params().Name {
		return nil, fmt.Errorf("APNs key curve must be P-256 for ES256, got %s", key.Curve.Params().Name)
	}

	return key, nil
}

func signES256(signingInput string, key *ecdsa.PrivateKey) ([]byte, error) {
	digest := sha256.Sum256([]byte(signingInput))
	r, sInt, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		return nil, fmt.Errorf("sign JWT with ES256: %w", err)
	}

	const scalarSize = 32
	signature := make([]byte, scalarSize*2)
	r.FillBytes(signature[:scalarSize])
	sInt.FillBytes(signature[scalarSize:])

	return signature, nil
}
