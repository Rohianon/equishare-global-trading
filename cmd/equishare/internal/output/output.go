package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/olekukonko/tablewriter"
)

var (
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	MutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	ValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	MoneyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))
)

func JSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func Table(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(true)
	table.SetRowLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("│")
	table.SetColumnSeparator("│")
	table.SetRowSeparator("─")
	table.SetHeaderLine(true)
	table.SetTablePadding(" ")
	table.AppendBulk(rows)
	table.Render()
}

func KeyValue(pairs [][]string) {
	maxKeyLen := 0
	for _, pair := range pairs {
		if len(pair[0]) > maxKeyLen {
			maxKeyLen = len(pair[0])
		}
	}

	for _, pair := range pairs {
		key := MutedStyle.Render(fmt.Sprintf("%-*s", maxKeyLen, pair[0]))
		value := ValueStyle.Render(pair[1])
		fmt.Printf("%s  %s\n", key, value)
	}
}

func Success(msg string) {
	fmt.Println(SuccessStyle.Render("✓ ") + msg)
}

func Error(msg string) {
	fmt.Fprintln(os.Stderr, ErrorStyle.Render("✗ ")+msg)
}

func Warning(msg string) {
	fmt.Println(WarningStyle.Render("⚠ ") + msg)
}

func Info(msg string) {
	fmt.Println(MutedStyle.Render(msg))
}

func Header(msg string) {
	fmt.Println(HeaderStyle.Render(msg))
}

func Money(amount float64, currency string) string {
	return MoneyStyle.Render(fmt.Sprintf("%s %.2f", currency, amount))
}

func FormatStatus(status string) string {
	switch status {
	case "completed", "success", "active":
		return SuccessStyle.Render(status)
	case "pending", "processing":
		return WarningStyle.Render(status)
	case "failed", "cancelled", "rejected":
		return ErrorStyle.Render(status)
	default:
		return status
	}
}
