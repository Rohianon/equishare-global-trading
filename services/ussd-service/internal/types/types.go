package types

import "time"

type USSDRequest struct {
	SessionID   string `form:"sessionId"`
	ServiceCode string `form:"serviceCode"`
	PhoneNumber string `form:"phoneNumber"`
	Text        string `form:"text"`
}

type Session struct {
	SessionID   string         `json:"session_id"`
	PhoneNumber string         `json:"phone_number"`
	State       string         `json:"state"`
	Data        map[string]any `json:"data"`
	UserID      string         `json:"user_id,omitempty"`
	Authenticated bool         `json:"authenticated"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func NewSession(sessionID, phoneNumber string) *Session {
	return &Session{
		SessionID:   sessionID,
		PhoneNumber: phoneNumber,
		State:       StateInit,
		Data:        make(map[string]any),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

const (
	StateInit         = "init"
	StateAuth         = "auth"
	StateMainMenu     = "main_menu"
	StateBuyMethod    = "buy.method"
	StateBuySearch    = "buy.search"
	StateBuySelect    = "buy.select"
	StateBuyAmount    = "buy.amount"
	StateBuyConfirm   = "buy.confirm"
	StateSellSelect   = "sell.select"
	StateSellQuantity = "sell.quantity"
	StateSellConfirm  = "sell.confirm"
	StatePortfolio    = "portfolio"
	StateDeposit      = "deposit"
	StateWithdraw     = "withdraw"
	StateWithdrawConfirm = "withdraw.confirm"
	StateComplete     = "complete"
)

type StateResponse struct {
	Response  string
	NextState string
	End       bool
}

func Continue(text string, nextState string) *StateResponse {
	return &StateResponse{
		Response:  "CON " + text,
		NextState: nextState,
		End:       false,
	}
}

func End(text string) *StateResponse {
	return &StateResponse{
		Response:  "END " + text,
		NextState: "",
		End:       true,
	}
}
