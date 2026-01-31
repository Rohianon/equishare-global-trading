package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/pkg/crypto"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/services/ussd-service/internal/session"
	"github.com/Rohianon/equishare-global-trading/services/ussd-service/internal/types"
)

type Handler struct {
	sessionMgr *session.Manager
	db         *pgxpool.Pool
}

func New(sessionMgr *session.Manager, db *pgxpool.Pool) *Handler {
	return &Handler{
		sessionMgr: sessionMgr,
		db:         db,
	}
}

func (h *Handler) Callback(c *fiber.Ctx) error {
	var req types.USSDRequest
	if err := c.BodyParser(&req); err != nil {
		logger.Error().Err(err).Msg("Failed to parse USSD request")
		return c.SendString("END System error. Please try again.")
	}

	logger.Info().
		Str("session_id", req.SessionID).
		Str("phone", req.PhoneNumber).
		Str("text", req.Text).
		Msg("USSD callback received")

	ctx := c.Context()

	sess, err := h.sessionMgr.GetOrCreate(ctx, req.SessionID, req.PhoneNumber)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get/create session")
		return c.SendString("END System error. Please try again.")
	}

	input := h.getLatestInput(req.Text)

	response := h.handleState(ctx, sess, input)

	if !response.End {
		sess.State = response.NextState
		if err := h.sessionMgr.Save(ctx, sess); err != nil {
			logger.Error().Err(err).Msg("Failed to save session")
		}
	} else {
		h.sessionMgr.Delete(ctx, sess.SessionID)
	}

	logger.Info().
		Str("session_id", req.SessionID).
		Str("state", sess.State).
		Str("next_state", response.NextState).
		Bool("end", response.End).
		Msg("USSD response sent")

	return c.SendString(response.Response)
}

func (h *Handler) getLatestInput(text string) string {
	if text == "" {
		return ""
	}
	parts := strings.Split(text, "*")
	return parts[len(parts)-1]
}

func (h *Handler) handleState(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	switch sess.State {
	case types.StateInit:
		return h.handleInit(ctx, sess, input)
	case types.StateAuth:
		return h.handleAuth(ctx, sess, input)
	case types.StateMainMenu:
		return h.handleMainMenu(ctx, sess, input)
	case types.StateBuyMethod:
		return h.handleBuyMethod(ctx, sess, input)
	case types.StateBuySearch:
		return h.handleBuySearch(ctx, sess, input)
	case types.StateBuySelect:
		return h.handleBuySelect(ctx, sess, input)
	case types.StateBuyAmount:
		return h.handleBuyAmount(ctx, sess, input)
	case types.StateBuyConfirm:
		return h.handleBuyConfirm(ctx, sess, input)
	case types.StateSellSelect:
		return h.handleSellSelect(ctx, sess, input)
	case types.StateSellQuantity:
		return h.handleSellQuantity(ctx, sess, input)
	case types.StateSellConfirm:
		return h.handleSellConfirm(ctx, sess, input)
	case types.StatePortfolio:
		return h.handlePortfolio(ctx, sess, input)
	case types.StateDeposit:
		return h.handleDeposit(ctx, sess, input)
	case types.StateWithdraw:
		return h.handleWithdraw(ctx, sess, input)
	case types.StateWithdrawConfirm:
		return h.handleWithdrawConfirm(ctx, sess, input)
	default:
		return types.End("Invalid session. Please dial again.")
	}
}

func (h *Handler) handleInit(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	user, err := h.getUserByPhone(ctx, sess.PhoneNumber)
	if err != nil || user == nil {
		return types.End("You are not registered. Please register via the app or SMS REGISTER to 40255.")
	}

	sess.UserID = user.ID
	sess.Data["user_id"] = user.ID

	return types.Continue("Welcome to EquiShare\nEnter your 4-digit PIN:", types.StateAuth)
}

func (h *Handler) handleAuth(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	if len(input) != 4 {
		return types.Continue("Invalid PIN. Enter your 4-digit PIN:", types.StateAuth)
	}

	user, err := h.getUserByID(ctx, sess.UserID)
	if err != nil || user == nil {
		return types.End("Authentication failed. Please try again.")
	}

	if user.PINHash == nil || !crypto.CheckPIN(input, *user.PINHash) {
		attempts, _ := sess.Data["auth_attempts"].(int)
		attempts++
		sess.Data["auth_attempts"] = attempts

		if attempts >= 3 {
			return types.End("Too many failed attempts. Please try again later.")
		}

		return types.Continue(fmt.Sprintf("Wrong PIN. %d attempt(s) remaining.\nEnter PIN:", 3-attempts), types.StateAuth)
	}

	sess.Authenticated = true
	return h.showMainMenu()
}

func (h *Handler) showMainMenu() *types.StateResponse {
	menu := `Welcome to EquiShare
1. Buy Shares
2. Sell Shares
3. My Portfolio
4. Deposit (M-Pesa)
5. Withdraw
0. Exit`
	return types.Continue(menu, types.StateMainMenu)
}

func (h *Handler) handleMainMenu(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	switch input {
	case "1":
		return types.Continue("Buy Shares\n1. Search by Name\n2. Popular Stocks\n0. Back", types.StateBuyMethod)
	case "2":
		return h.handleSellSelect(ctx, sess, "")
	case "3":
		return h.handlePortfolio(ctx, sess, "")
	case "4":
		return types.Continue("Enter deposit amount (KES):", types.StateDeposit)
	case "5":
		return types.Continue("Enter withdrawal amount (KES):", types.StateWithdraw)
	case "0":
		return types.End("Thank you for using EquiShare. Goodbye!")
	default:
		return h.showMainMenu()
	}
}

func (h *Handler) handleBuyMethod(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	switch input {
	case "1":
		return types.Continue("Enter stock name or symbol:", types.StateBuySearch)
	case "2":
		sess.Data["stocks"] = []string{"AAPL", "GOOGL", "MSFT", "AMZN", "TSLA"}
		return types.Continue("Popular Stocks:\n1. AAPL - Apple\n2. GOOGL - Google\n3. MSFT - Microsoft\n4. AMZN - Amazon\n5. TSLA - Tesla\n0. Back", types.StateBuySelect)
	case "0":
		return h.showMainMenu()
	default:
		return types.Continue("Invalid option.\n1. Search by Name\n2. Popular Stocks\n0. Back", types.StateBuyMethod)
	}
}

func (h *Handler) handleBuySearch(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	if input == "0" {
		return types.Continue("Buy Shares\n1. Search by Name\n2. Popular Stocks\n0. Back", types.StateBuyMethod)
	}

	searchResults := []string{"AAPL", "GOOGL"}
	sess.Data["stocks"] = searchResults
	sess.Data["search_term"] = input

	return types.Continue(fmt.Sprintf("Results for '%s':\n1. AAPL - Apple ($150.00)\n2. GOOGL - Google ($140.00)\n0. Back", input), types.StateBuySelect)
}

func (h *Handler) handleBuySelect(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	if input == "0" {
		return types.Continue("Buy Shares\n1. Search by Name\n2. Popular Stocks\n0. Back", types.StateBuyMethod)
	}

	stocks, _ := sess.Data["stocks"].([]string)
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(stocks) {
		return types.Continue("Invalid selection. Please try again:", types.StateBuySelect)
	}

	selectedStock := stocks[idx-1]
	sess.Data["selected_stock"] = selectedStock

	return types.Continue(fmt.Sprintf("Buy %s\nEnter amount in KES:", selectedStock), types.StateBuyAmount)
}

func (h *Handler) handleBuyAmount(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	if input == "0" {
		return h.handleBuyMethod(ctx, sess, "2")
	}

	amount, err := strconv.ParseFloat(input, 64)
	if err != nil || amount < 100 {
		return types.Continue("Invalid amount. Minimum KES 100.\nEnter amount in KES:", types.StateBuyAmount)
	}

	sess.Data["amount"] = amount
	stock := sess.Data["selected_stock"].(string)

	return types.Continue(fmt.Sprintf("Confirm purchase:\nStock: %s\nAmount: KES %.2f\n\n1. Confirm\n2. Cancel", stock, amount), types.StateBuyConfirm)
}

func (h *Handler) handleBuyConfirm(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	switch input {
	case "1":
		stock := sess.Data["selected_stock"].(string)
		amount := sess.Data["amount"].(float64)
		return types.End(fmt.Sprintf("Order placed!\nBuying %s for KES %.2f\nYou will receive SMS confirmation.", stock, amount))
	case "2":
		return h.showMainMenu()
	default:
		return types.Continue("1. Confirm\n2. Cancel", types.StateBuyConfirm)
	}
}

func (h *Handler) handleSellSelect(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	if input == "" {
		holdings := []string{"AAPL (5 shares)", "GOOGL (3 shares)"}
		sess.Data["holdings"] = holdings

		if len(holdings) == 0 {
			return types.Continue("You have no shares to sell.\n\n0. Back to Menu", types.StateMainMenu)
		}

		menu := "Your Holdings:\n"
		for i, h := range holdings {
			menu += fmt.Sprintf("%d. %s\n", i+1, h)
		}
		menu += "0. Back"
		return types.Continue(menu, types.StateSellSelect)
	}

	if input == "0" {
		return h.showMainMenu()
	}

	holdings, _ := sess.Data["holdings"].([]string)
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(holdings) {
		return types.Continue("Invalid selection. Please try again:", types.StateSellSelect)
	}

	sess.Data["sell_holding"] = holdings[idx-1]
	return types.Continue("Enter quantity to sell:", types.StateSellQuantity)
}

func (h *Handler) handleSellQuantity(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	if input == "0" {
		return h.handleSellSelect(ctx, sess, "")
	}

	qty, err := strconv.Atoi(input)
	if err != nil || qty < 1 {
		return types.Continue("Invalid quantity. Enter a valid number:", types.StateSellQuantity)
	}

	sess.Data["sell_quantity"] = qty
	holding := sess.Data["sell_holding"].(string)

	return types.Continue(fmt.Sprintf("Confirm sale:\n%s\nQuantity: %d\n\n1. Confirm\n2. Cancel", holding, qty), types.StateSellConfirm)
}

func (h *Handler) handleSellConfirm(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	switch input {
	case "1":
		holding := sess.Data["sell_holding"].(string)
		qty := sess.Data["sell_quantity"].(int)
		return types.End(fmt.Sprintf("Sell order placed!\nSelling %d of %s\nYou will receive SMS confirmation.", qty, holding))
	case "2":
		return h.showMainMenu()
	default:
		return types.Continue("1. Confirm\n2. Cancel", types.StateSellConfirm)
	}
}

func (h *Handler) handlePortfolio(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	if input == "0" || input != "" {
		return h.showMainMenu()
	}

	portfolio := `Your Portfolio:
-----------------
AAPL: 5 shares ($750)
GOOGL: 3 shares ($420)
-----------------
Total: $1,170

0. Back to Menu`

	return types.Continue(portfolio, types.StatePortfolio)
}

func (h *Handler) handleDeposit(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	if input == "0" {
		return h.showMainMenu()
	}

	amount, err := strconv.ParseFloat(input, 64)
	if err != nil || amount < 10 || amount > 150000 {
		return types.Continue("Invalid amount. Enter KES 10 - 150,000:", types.StateDeposit)
	}

	return types.End(fmt.Sprintf("Deposit initiated!\nAmount: KES %.2f\nYou will receive an M-Pesa prompt shortly.", amount))
}

func (h *Handler) handleWithdraw(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	if input == "0" {
		return h.showMainMenu()
	}

	amount, err := strconv.ParseFloat(input, 64)
	if err != nil || amount < 10 {
		return types.Continue("Invalid amount. Minimum KES 10.\nEnter amount:", types.StateWithdraw)
	}

	sess.Data["withdraw_amount"] = amount
	return types.Continue(fmt.Sprintf("Confirm withdrawal:\nAmount: KES %.2f to %s\n\n1. Confirm\n2. Cancel", amount, sess.PhoneNumber), types.StateWithdrawConfirm)
}

func (h *Handler) handleWithdrawConfirm(ctx context.Context, sess *types.Session, input string) *types.StateResponse {
	switch input {
	case "1":
		amount := sess.Data["withdraw_amount"].(float64)
		return types.End(fmt.Sprintf("Withdrawal initiated!\nKES %.2f will be sent to %s", amount, sess.PhoneNumber))
	case "2":
		return h.showMainMenu()
	default:
		return types.Continue("1. Confirm\n2. Cancel", types.StateWithdrawConfirm)
	}
}

type User struct {
	ID       string
	Phone    string
	PINHash  *string
	IsActive bool
}

func (h *Handler) getUserByPhone(ctx context.Context, phone string) (*User, error) {
	var user User
	err := h.db.QueryRow(ctx, `
		SELECT id, phone, pin_hash, is_active FROM users WHERE phone = $1
	`, phone).Scan(&user.ID, &user.Phone, &user.PINHash, &user.IsActive)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (h *Handler) getUserByID(ctx context.Context, id string) (*User, error) {
	var user User
	err := h.db.QueryRow(ctx, `
		SELECT id, phone, pin_hash, is_active FROM users WHERE id = $1
	`, id).Scan(&user.ID, &user.Phone, &user.PINHash, &user.IsActive)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
