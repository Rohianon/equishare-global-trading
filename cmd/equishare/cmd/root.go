package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	format  string

	// Styles
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
)

var rootCmd = &cobra.Command{
	Use:   "equishare",
	Short: "EquiShare - Trade global stocks with M-Pesa",
	Long: titleStyle.Render(`
╔═══════════════════════════════════════════════════════════╗
║  EquiShare CLI - Trade Global Stocks with M-Pesa         ║
╚═══════════════════════════════════════════════════════════╝
`) + `
Access global stock markets from your terminal.
Deposit via M-Pesa, trade fractional shares, and manage your portfolio.

Get started:
  equishare auth register    Register a new account
  equishare auth login       Login to your account
  equishare wallet balance   Check your balance
  equishare --help           Show all commands`,
	Version: "1.0.0",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.equishare/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "table", "output format: table, json")

	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render("Error: ")+err.Error())
			os.Exit(1)
		}

		configDir := filepath.Join(home, ".equishare")
		if err := os.MkdirAll(configDir, 0700); err != nil {
			fmt.Fprintln(os.Stderr, errorStyle.Render("Error creating config dir: ")+err.Error())
		}

		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Set defaults
	viper.SetDefault("api_url", "http://localhost:8000")
	viper.SetDefault("format", "table")
	viper.SetDefault("currency", "KES")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		// Config loaded
	}
}

func getFormat() string {
	if format != "" && format != "table" {
		return format
	}
	return viper.GetString("format")
}

func printSuccess(msg string) {
	fmt.Println(successStyle.Render("✓ ") + msg)
}

func printError(msg string) {
	fmt.Fprintln(os.Stderr, errorStyle.Render("✗ ")+msg)
}

func printInfo(msg string) {
	fmt.Println(infoStyle.Render(msg))
}
