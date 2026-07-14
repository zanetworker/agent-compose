package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"github.com/zanetworker/agent-compose/pkg/compose"
)

func initCmd() *cobra.Command {
	var skipProviders bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create ~/.ac/ with default config and auto-create providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, _ := os.UserHomeDir()
			dir := filepath.Join(home, ".ac")
			if err := os.MkdirAll(filepath.Join(dir, "skills"), 0755); err != nil {
				return err
			}

			cfgPath := filepath.Join(dir, "config.yaml")
			if _, err := os.Stat(cfgPath); err == nil {
				fmt.Fprintf(os.Stderr, "Config already exists at %s\n", cfgPath)
			} else {
				cfg := compose.DefaultConfig()
				data, err := yaml.Marshal(cfg)
				if err != nil {
					return err
				}
				if err := os.WriteFile(cfgPath, data, 0644); err != nil {
					return err
				}
				fmt.Printf("Created %s\n", cfgPath)
			}

			if skipProviders {
				return nil
			}

			fmt.Println("\nDetecting local credentials...")
			openshellBin := "openshell"
			created := 0

			// Google Cloud / Vertex AI
			adcPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
			if adcPath == "" {
				adcPath = filepath.Join(home, ".config", "gcloud", "application_default_credentials.json")
			}
			if _, err := os.Stat(adcPath); err == nil {
				// Vertex AI provider (inference)
				if providerExists(openshellBin, "vertex") {
					fmt.Println("  Google Cloud ADC found       vertex provider already exists")
				} else {
					out, err := exec.Command(openshellBin, "provider", "create",
						"--type", "google-vertex-ai",
						"--name", "vertex",
						"--from-gcloud-adc").CombinedOutput()
					if err != nil {
						fmt.Fprintf(os.Stderr, "  Google Cloud ADC found       failed to create vertex provider: %s\n", strings.TrimSpace(string(out)))
					} else {
						fmt.Println("  Google Cloud ADC found       created vertex provider")
						created++
					}
				}

				// Google Cloud provider (metadata emulator for GCP SDK auth)
				if providerExists(openshellBin, "gcp") {
					fmt.Println("  Google Cloud ADC found       gcp provider already exists")
				} else {
					if err := createGCPProvider(openshellBin, adcPath); err != nil {
						fmt.Fprintf(os.Stderr, "  Google Cloud ADC found       failed to create gcp provider: %v\n", err)
					} else {
						fmt.Println("  Google Cloud ADC found       created gcp provider (metadata emulator)")
						created++
					}
				}
			} else {
				fmt.Println("  Google Cloud ADC             not found (run: gcloud auth application-default login)")
			}

			// GitHub
			ghToken, err := exec.Command("gh", "auth", "token").Output()
			if err == nil && len(strings.TrimSpace(string(ghToken))) > 0 {
				if providerExists(openshellBin, "github") {
					fmt.Println("  GitHub token found           github provider already exists")
				} else {
					cmd := exec.Command(openshellBin, "provider", "create",
						"--type", "github",
						"--name", "github",
						"--credential", "api_token")
					cmd.Env = append(os.Environ(), "api_token="+strings.TrimSpace(string(ghToken)))
					out, err := cmd.CombinedOutput()
					if err != nil {
						fmt.Fprintf(os.Stderr, "  GitHub token found           failed to create github provider: %s\n", strings.TrimSpace(string(out)))
					} else {
						fmt.Println("  GitHub token found           created github provider")
						created++
					}
				}
			} else {
				fmt.Println("  GitHub token                 not found (run: gh auth login)")
			}

			// Anthropic API key
			if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
				if providerExists(openshellBin, "claude-code") {
					fmt.Println("  Anthropic API key found      claude-code provider already exists")
				} else {
					cmd := exec.Command(openshellBin, "provider", "create",
						"--type", "claude-code",
						"--name", "claude-code",
						"--credential", "api_key")
					cmd.Env = append(os.Environ(), "api_key="+key)
					out, err := cmd.CombinedOutput()
					if err != nil {
						fmt.Fprintf(os.Stderr, "  Anthropic API key found      failed to create claude-code provider: %s\n", strings.TrimSpace(string(out)))
					} else {
						fmt.Println("  Anthropic API key found      created claude-code provider")
						created++
					}
				}
			} else {
				fmt.Println("  Anthropic API key            not set (using Vertex? That's fine)")
			}

			if created > 0 {
				fmt.Printf("\nCreated %d provider(s).\n", created)
			} else {
				fmt.Println("\nNo new providers created.")
			}

			fmt.Printf("\nTry:  ac run --runtime claude-code-vertex --prompt \"Hello\" --dry-run\n")
			return nil
		},
	}

	cmd.Flags().BoolVar(&skipProviders, "skip-providers", false, "skip auto-creating providers")
	return cmd
}

func createGCPProvider(bin, adcPath string) error {
	data, err := os.ReadFile(adcPath)
	if err != nil {
		return fmt.Errorf("reading ADC file: %w", err)
	}

	type adcJSON struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RefreshToken string `json:"refresh_token"`
	}
	var adc adcJSON
	if err := json.Unmarshal(data, &adc); err != nil {
		return fmt.Errorf("parsing ADC JSON: %w", err)
	}
	if adc.ClientID == "" || adc.ClientSecret == "" || adc.RefreshToken == "" {
		return fmt.Errorf("ADC file missing client_id, client_secret, or refresh_token")
	}

	// Step 1: create the provider with a placeholder credential
	env := append(os.Environ(), "GCP_ADC_ACCESS_TOKEN=placeholder")
	cmd := exec.Command(bin, "provider", "create",
		"--type", "google-cloud",
		"--name", "gcp",
		"--credential", "GCP_ADC_ACCESS_TOKEN")
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("creating provider: %s", strings.TrimSpace(string(out)))
	}

	// Step 2: configure refresh with ADC material
	out, err := exec.Command(bin, "provider", "refresh", "configure", "gcp",
		"--credential-key", "GCP_ADC_ACCESS_TOKEN",
		"--strategy", "oauth2-refresh-token",
		"--material", "client_id="+adc.ClientID,
		"--material", "client_secret="+adc.ClientSecret,
		"--material", "refresh_token="+adc.RefreshToken,
		"--secret-material-key", "client_secret",
		"--secret-material-key", "refresh_token",
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("configuring refresh: %s", strings.TrimSpace(string(out)))
	}

	return nil
}

func providerExists(bin, name string) bool {
	out, err := exec.Command(bin, "provider", "list").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] == name {
			return true
		}
	}
	return false
}
