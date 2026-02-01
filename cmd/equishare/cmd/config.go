package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Rohianon/equishare-global-trading/cmd/equishare/internal/output"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration commands",
	Long:  "View and modify CLI configuration.",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  "Display all current configuration values.",
	RunE:  runConfigShow,
}

var configSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  api_url   - API server URL (default: http://localhost:8000)
  format    - Default output format: table, json (default: table)
  currency  - Display currency (default: KES)`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show config file path",
	Long:  "Display the path to the configuration file.",
	RunE:  runConfigPath,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	settings := map[string]interface{}{
		"api_url":  viper.GetString("api_url"),
		"format":   viper.GetString("format"),
		"currency": viper.GetString("currency"),
	}

	if getFormat() == "json" {
		return output.JSON(settings)
	}

	output.Header("Configuration")
	fmt.Println()
	output.KeyValue([][]string{
		{"api_url", viper.GetString("api_url")},
		{"format", viper.GetString("format")},
		{"currency", viper.GetString("currency")},
	})

	if viper.ConfigFileUsed() != "" {
		fmt.Println()
		output.Info("Config file: " + viper.ConfigFileUsed())
	}

	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	// Validate key
	validKeys := map[string]bool{
		"api_url":  true,
		"format":   true,
		"currency": true,
	}

	if !validKeys[key] {
		output.Error(fmt.Sprintf("Unknown config key: %s", key))
		output.Info("Valid keys: api_url, format, currency")
		return nil
	}

	// Validate format value
	if key == "format" && value != "table" && value != "json" {
		output.Error("format must be 'table' or 'json'")
		return nil
	}

	viper.Set(key, value)

	// Ensure config directory exists
	home, err := os.UserHomeDir()
	if err != nil {
		output.Error("Could not find home directory: " + err.Error())
		return nil
	}

	configDir := filepath.Join(home, ".equishare")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		output.Error("Could not create config directory: " + err.Error())
		return nil
	}

	configFile := filepath.Join(configDir, "config.yaml")

	if err := viper.WriteConfigAs(configFile); err != nil {
		output.Error("Could not save config: " + err.Error())
		return nil
	}

	output.Success(fmt.Sprintf("Set %s = %s", key, value))
	return nil
}

func runConfigPath(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		output.Error("Could not find home directory: " + err.Error())
		return nil
	}

	configFile := filepath.Join(home, ".equishare", "config.yaml")

	if getFormat() == "json" {
		return output.JSON(map[string]string{
			"config_file": configFile,
			"config_dir":  filepath.Dir(configFile),
		})
	}

	fmt.Println(configFile)
	return nil
}
