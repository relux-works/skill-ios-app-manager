package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/relux-works/ios-app-manager/internal/config"
	"github.com/relux-works/ios-app-manager/internal/push"
	"github.com/spf13/cobra"
)

type pushSender interface {
	Send(ctx context.Context, request push.SendRequest) (push.SendResponse, error)
}

type pushTokenExtractor interface {
	LatestDeviceToken(ctx context.Context, fallbackTokenFilePath string) (string, error)
}

var newPushSender = func(cfg push.SenderConfig) pushSender {
	return push.NewSender(cfg)
}

var newPushTokenExtractor = func() pushTokenExtractor {
	return push.NewTokenExtractor()
}

func newPushCommand(opts *RootOptions) *cobra.Command {
	configPath := config.DefaultConfigPath
	if opts != nil && strings.TrimSpace(opts.ConfigPath) != "" {
		configPath = strings.TrimSpace(opts.ConfigPath)
	}

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push notification tools",
	}

	cmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		configPath,
		"Path to project config JSON file",
	)

	var deviceToken string
	var envValue string
	var payloadPath string
	var tokenFilePath string

	tokenCommand := &cobra.Command{
		Use:   "token",
		Short: "Print the latest APNs device token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("push token does not accept positional arguments")
			}

			fallbackPath, err := resolvePushTokenFallbackPath(tokenFilePath, configPath, opts)
			if err != nil {
				return err
			}

			token, err := newPushTokenExtractor().LatestDeviceToken(cmd.Context(), fallbackPath)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), token)
			return err
		},
	}
	tokenCommand.Flags().StringVar(
		&tokenFilePath,
		"token-file",
		"",
		"Fallback file path containing APNs token (for unreliable simulator logs)",
	)

	sendCommand := &cobra.Command{
		Use:   "send --token <device_token>",
		Short: "Send push notification via APNs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("push send does not accept positional arguments")
			}

			environment, err := push.ParseEnvironment(envValue)
			if err != nil {
				return err
			}

			selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
			cfg, err := config.LoadConfig(selectedConfigPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			senderConfig, err := buildPushSenderConfig(cfg, selectedConfigPath)
			if err != nil {
				return err
			}

			customPayload, err := loadPushPayloadFile(payloadPath)
			if err != nil {
				return err
			}

			sender := newPushSender(senderConfig)
			response, err := sender.Send(cmd.Context(), push.SendRequest{
				DeviceToken: deviceToken,
				Environment: environment,
				AppName:     cfg.AppName,
				Payload:     customPayload,
			})
			if err != nil {
				return fmt.Errorf("send push: %w", err)
			}

			if response.ErrorDescription == "" {
				if _, err := fmt.Fprintf(
					cmd.OutOrStdout(),
					"APNs response: status=%d apns-id=%s\n",
					response.StatusCode,
					response.APNSID,
				); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintf(
					cmd.OutOrStdout(),
					"APNs response: status=%d apns-id=%s reason=%s\n",
					response.StatusCode,
					response.APNSID,
					response.ErrorDescription,
				); err != nil {
					return err
				}
			}

			if response.StatusCode < 200 || response.StatusCode >= 300 {
				if response.ErrorDescription == "" {
					return fmt.Errorf("APNs returned status %d", response.StatusCode)
				}
				return fmt.Errorf(
					"APNs returned status %d: %s",
					response.StatusCode,
					response.ErrorDescription,
				)
			}

			return nil
		},
	}

	sendCommand.Flags().StringVar(
		&deviceToken,
		"token",
		"",
		"APNs device token",
	)
	sendCommand.Flags().StringVar(
		&envValue,
		"env",
		"dev",
		"APNs environment (dev|prod)",
	)
	sendCommand.Flags().StringVar(
		&payloadPath,
		"payload",
		"",
		"Path to JSON payload file",
	)
	_ = sendCommand.MarkFlagRequired("token")

	cmd.AddCommand(tokenCommand, sendCommand)

	return cmd
}

func buildPushSenderConfig(cfg config.ProjectConfig, configPath string) (push.SenderConfig, error) {
	keyPath := strings.TrimSpace(cfg.PushKeyPath)
	if keyPath == "" {
		return push.SenderConfig{}, fmt.Errorf("push_key_path is required in config for push send")
	}

	if !filepath.IsAbs(keyPath) {
		keyPath = filepath.Join(filepath.Dir(configPath), keyPath)
	}

	keyID := strings.TrimSpace(cfg.PushKeyID)
	if keyID == "" {
		return push.SenderConfig{}, fmt.Errorf("push_key_id is required in config for push send")
	}

	teamID := strings.TrimSpace(cfg.TeamID)
	if teamID == "" {
		return push.SenderConfig{}, fmt.Errorf("team_id is required in config for push send")
	}

	bundleID := strings.TrimSpace(cfg.BundleID)
	if bundleID == "" {
		return push.SenderConfig{}, fmt.Errorf("bundle_id is required in config for push send")
	}

	return push.SenderConfig{
		KeyPath:  filepath.Clean(keyPath),
		KeyID:    keyID,
		TeamID:   teamID,
		BundleID: bundleID,
	}, nil
}

func loadPushPayloadFile(payloadPath string) ([]byte, error) {
	path := strings.TrimSpace(payloadPath)
	if path == "" {
		return nil, nil
	}

	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read payload file %q: %w", path, err)
	}

	return content, nil
}

func resolvePushTokenFallbackPath(tokenFilePath string, configPath string, opts *RootOptions) (string, error) {
	explicitPath := strings.TrimSpace(tokenFilePath)
	if explicitPath != "" {
		return filepath.Clean(explicitPath), nil
	}

	selectedConfigPath := resolveSelectedConfigPath(configPath, opts)
	if strings.TrimSpace(selectedConfigPath) == "" {
		return "", nil
	}

	info, err := os.Stat(selectedConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}

		return "", fmt.Errorf("stat config %q: %w", selectedConfigPath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("config path %q is a directory", selectedConfigPath)
	}

	cfg, err := config.LoadConfig(selectedConfigPath)
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}

	return strings.TrimSpace(cfg.PushTokenPath), nil
}
