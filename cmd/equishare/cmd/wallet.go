package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/Rohianon/equishare-global-trading/cmd/equishare/internal/auth"
	"github.com/Rohianon/equishare-global-trading/cmd/equishare/internal/client"
	"github.com/Rohianon/equishare-global-trading/cmd/equishare/internal/output"
)

var walletCmd = &cobra.Command{
	Use:   "wallet",
	Short: "Wallet and payment commands",
	Long:  "Manage your wallet - check balance, deposit via M-Pesa, view transactions.",
}

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Show wallet balance",
	Long:  "Display your current wallet balance.",
	RunE:  runBalance,
}

var depositCmd = &cobra.Command{
	Use:   "deposit",
	Short: "Deposit via M-Pesa",
	Long:  "Initiate an M-Pesa STK push to deposit funds into your wallet.",
	RunE:  runDeposit,
}

var transactionsCmd = &cobra.Command{
	Use:   "transactions",
	Short: "View transaction history",
	Long:  "List your recent wallet transactions.",
	RunE:  runTransactions,
}

var (
	amountFlag float64
	pageFlag   int
	limitFlag  int
)

func init() {
	rootCmd.AddCommand(walletCmd)
	walletCmd.AddCommand(balanceCmd)
	walletCmd.AddCommand(depositCmd)
	walletCmd.AddCommand(transactionsCmd)

	depositCmd.Flags().Float64VarP(&amountFlag, "amount", "a", 0, "amount to deposit (KES)")
	depositCmd.Flags().StringVarP(&phoneFlag, "phone", "p", "", "M-Pesa phone number (defaults to account phone)")

	transactionsCmd.Flags().IntVar(&pageFlag, "page", 1, "page number")
	transactionsCmd.Flags().IntVar(&limitFlag, "limit", 10, "items per page")
}

func requireAuth() (*client.Client, error) {
	if !auth.IsLoggedIn() {
		output.Error("Not logged in. Run 'equishare auth login' first.")
		return nil, fmt.Errorf("not authenticated")
	}

	c := client.New()
	c.SetToken(auth.GetToken())
	return c, nil
}

func runBalance(cmd *cobra.Command, args []string) error {
	c, err := requireAuth()
	if err != nil {
		return nil
	}

	balance, err := c.GetWalletBalance()
	if err != nil {
		output.Error(err.Error())
		return nil
	}

	if getFormat() == "json" {
		return output.JSON(balance)
	}

	output.Header("Wallet Balance")
	fmt.Println()
	output.KeyValue([][]string{
		{"Available", output.Money(balance.Available, balance.Currency)},
		{"Pending", output.Money(balance.Pending, balance.Currency)},
		{"Total", output.Money(balance.Total, balance.Currency)},
	})

	return nil
}

func runDeposit(cmd *cobra.Command, args []string) error {
	c, err := requireAuth()
	if err != nil {
		return nil
	}

	amount := amountFlag
	if amount <= 0 {
		amountStr := prompt("Amount (KES)")
		amount, err = strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			output.Error("Invalid amount")
			return nil
		}
	}

	phone := phoneFlag
	if phone == "" {
		stored, _ := auth.Load()
		if stored != nil && stored.Phone != "" {
			phone = stored.Phone
		} else {
			phone = prompt("M-Pesa phone number")
		}
	}
	phone = normalizePhone(phone)

	output.Info(fmt.Sprintf("Initiating M-Pesa deposit of KES %.2f...", amount))

	resp, err := c.InitiateDeposit(amount, phone)
	if err != nil {
		output.Error(err.Error())
		return nil
	}

	if getFormat() == "json" {
		return output.JSON(resp)
	}

	output.Success("STK push sent!")
	fmt.Println()
	output.Info("Check your phone for the M-Pesa prompt")
	output.Info("Enter your M-Pesa PIN to complete the transaction")
	fmt.Println()
	output.KeyValue([][]string{
		{"Transaction ID", resp.TransactionID},
		{"Status", resp.Status},
	})

	return nil
}

func runTransactions(cmd *cobra.Command, args []string) error {
	c, err := requireAuth()
	if err != nil {
		return nil
	}

	resp, err := c.GetTransactions(pageFlag, limitFlag)
	if err != nil {
		output.Error(err.Error())
		return nil
	}

	if getFormat() == "json" {
		return output.JSON(resp)
	}

	if len(resp.Transactions) == 0 {
		output.Info("No transactions found")
		return nil
	}

	output.Header("Transaction History")
	fmt.Println()

	rows := make([][]string, len(resp.Transactions))
	for i, tx := range resp.Transactions {
		rows[i] = []string{
			tx.ID[:8],
			tx.Type,
			fmt.Sprintf("%.2f %s", tx.Amount, tx.Currency),
			output.FormatStatus(tx.Status),
			tx.CreatedAt[:10],
		}
	}

	output.Table([]string{"ID", "Type", "Amount", "Status", "Date"}, rows)
	fmt.Println()
	output.Info(fmt.Sprintf("Page %d of %d", resp.Page, (resp.Total+resp.PerPage-1)/resp.PerPage))

	return nil
}
