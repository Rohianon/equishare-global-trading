package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Rohianon/equishare-global-trading/cmd/equishare/internal/auth"
	"github.com/Rohianon/equishare-global-trading/cmd/equishare/internal/client"
	"github.com/Rohianon/equishare-global-trading/cmd/equishare/internal/output"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  "Manage your EquiShare account authentication - register, login, logout.",
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new account",
	Long:  "Register a new EquiShare account with your phone number.",
	RunE:  runRegister,
}

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify OTP and set PIN",
	Long:  "Verify your phone number with the OTP sent via SMS and set your PIN.",
	RunE:  runVerify,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to your account",
	Long:  "Login to your EquiShare account with phone and PIN.",
	RunE:  runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout from your account",
	Long:  "Logout and clear stored credentials.",
	RunE:  runLogout,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  "Show current authentication status and user info.",
	RunE:  runStatus,
}

var (
	phoneFlag    string
	fullNameFlag string
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(registerCmd)
	authCmd.AddCommand(verifyCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)

	registerCmd.Flags().StringVarP(&phoneFlag, "phone", "p", "", "phone number (e.g., 0712345678)")
	registerCmd.Flags().StringVarP(&fullNameFlag, "name", "n", "", "full name")

	verifyCmd.Flags().StringVarP(&phoneFlag, "phone", "p", "", "phone number")

	loginCmd.Flags().StringVarP(&phoneFlag, "phone", "p", "", "phone number")
}

func runRegister(cmd *cobra.Command, args []string) error {
	phone := phoneFlag
	if phone == "" {
		phone = prompt("Phone number")
	}
	phone = normalizePhone(phone)

	name := fullNameFlag
	if name == "" {
		name = prompt("Full name (optional)")
	}

	c := client.New()
	output.Info("Registering account...")

	resp, err := c.Register(phone, name)
	if err != nil {
		output.Error(err.Error())
		return nil
	}

	output.Success("Registration initiated!")
	fmt.Println()
	output.Info(fmt.Sprintf("An OTP has been sent to %s", phone))
	output.Info(fmt.Sprintf("The code expires in %d seconds", resp.ExpiresIn))
	fmt.Println()
	output.Info("Run 'equishare auth verify --phone " + phone + "' to complete registration")

	return nil
}

func runVerify(cmd *cobra.Command, args []string) error {
	phone := phoneFlag
	if phone == "" {
		phone = prompt("Phone number")
	}
	phone = normalizePhone(phone)

	code := prompt("OTP code")
	pin := promptSecret("Set your PIN (4-6 digits)")

	c := client.New()
	output.Info("Verifying...")

	resp, err := c.Verify(phone, code, pin)
	if err != nil {
		output.Error(err.Error())
		return nil
	}

	// Save auth
	expiresAt := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	if err := auth.Save(&auth.StoredAuth{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresAt:    expiresAt,
		Phone:        resp.User.Phone,
		UserID:       resp.User.ID,
		FullName:     resp.User.FullName,
	}); err != nil {
		output.Warning("Could not save credentials: " + err.Error())
	}

	output.Success("Account verified and logged in!")
	fmt.Println()
	output.KeyValue([][]string{
		{"User ID", resp.User.ID},
		{"Phone", resp.User.Phone},
		{"Name", resp.User.FullName},
	})

	return nil
}

func runLogin(cmd *cobra.Command, args []string) error {
	phone := phoneFlag
	if phone == "" {
		phone = prompt("Phone number")
	}
	phone = normalizePhone(phone)

	pin := promptSecret("PIN")

	c := client.New()
	output.Info("Logging in...")

	resp, err := c.Login(phone, pin)
	if err != nil {
		output.Error(err.Error())
		return nil
	}

	// Save auth
	expiresAt := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	if err := auth.Save(&auth.StoredAuth{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresAt:    expiresAt,
		Phone:        resp.User.Phone,
		UserID:       resp.User.ID,
		FullName:     resp.User.FullName,
	}); err != nil {
		output.Warning("Could not save credentials: " + err.Error())
	}

	output.Success("Logged in successfully!")
	fmt.Println()
	output.KeyValue([][]string{
		{"User", resp.User.FullName},
		{"Phone", resp.User.Phone},
	})

	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	if err := auth.Clear(); err != nil {
		output.Error("Failed to logout: " + err.Error())
		return nil
	}

	output.Success("Logged out successfully")
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	stored, err := auth.Load()
	if err != nil {
		output.Error("Failed to read auth: " + err.Error())
		return nil
	}

	if stored == nil || stored.AccessToken == "" {
		if getFormat() == "json" {
			return output.JSON(map[string]interface{}{
				"logged_in": false,
			})
		}
		output.Info("Not logged in")
		output.Info("Run 'equishare auth login' to login")
		return nil
	}

	isExpired := time.Now().After(stored.ExpiresAt)

	if getFormat() == "json" {
		return output.JSON(map[string]interface{}{
			"logged_in":  !isExpired,
			"user_id":    stored.UserID,
			"phone":      stored.Phone,
			"full_name":  stored.FullName,
			"expires_at": stored.ExpiresAt,
			"expired":    isExpired,
		})
	}

	if isExpired {
		output.Warning("Session expired")
		output.Info("Run 'equishare auth login' to login again")
		return nil
	}

	output.Success("Logged in")
	fmt.Println()
	output.KeyValue([][]string{
		{"User ID", stored.UserID},
		{"Phone", stored.Phone},
		{"Name", stored.FullName},
		{"Expires", stored.ExpiresAt.Format(time.RFC3339)},
	})

	return nil
}

func prompt(label string) string {
	fmt.Printf("%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func promptSecret(label string) string {
	fmt.Printf("%s: ", label)
	bytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return ""
	}
	return string(bytes)
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")

	// Convert 07... to +2547...
	if strings.HasPrefix(phone, "07") {
		phone = "+254" + phone[1:]
	} else if strings.HasPrefix(phone, "01") {
		phone = "+254" + phone[1:]
	} else if strings.HasPrefix(phone, "254") {
		phone = "+" + phone
	}

	return phone
}
