package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Rohianon/equishare-global-trading/services/payment-service/internal/types"
)

type MpesaRepository struct {
	db *pgxpool.Pool
}

func NewMpesaRepository(db *pgxpool.Pool) *MpesaRepository {
	return &MpesaRepository{db: db}
}

func (r *MpesaRepository) Create(ctx context.Context, userID, checkoutRequestID, merchantRequestID, phone string, amount float64) (*types.MpesaTransaction, error) {
	var tx types.MpesaTransaction

	err := r.db.QueryRow(ctx, `
		INSERT INTO mpesa_transactions (user_id, checkout_request_id, merchant_request_id, phone, amount, status)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		RETURNING id, user_id, transaction_id, checkout_request_id, merchant_request_id,
		          amount, phone, status, mpesa_receipt, result_code, result_desc,
		          callback_payload, created_at, updated_at
	`, userID, checkoutRequestID, merchantRequestID, phone, amount).Scan(
		&tx.ID, &tx.UserID, &tx.TransactionID, &tx.CheckoutRequestID, &tx.MerchantRequestID,
		&tx.Amount, &tx.Phone, &tx.Status, &tx.MpesaReceipt, &tx.ResultCode, &tx.ResultDesc,
		&tx.CallbackPayload, &tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mpesa transaction: %w", err)
	}

	return &tx, nil
}

func (r *MpesaRepository) GetByCheckoutRequestID(ctx context.Context, checkoutRequestID string) (*types.MpesaTransaction, error) {
	var tx types.MpesaTransaction

	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, transaction_id, checkout_request_id, merchant_request_id,
		       amount, phone, status, mpesa_receipt, result_code, result_desc,
		       callback_payload, created_at, updated_at
		FROM mpesa_transactions WHERE checkout_request_id = $1
	`, checkoutRequestID).Scan(
		&tx.ID, &tx.UserID, &tx.TransactionID, &tx.CheckoutRequestID, &tx.MerchantRequestID,
		&tx.Amount, &tx.Phone, &tx.Status, &tx.MpesaReceipt, &tx.ResultCode, &tx.ResultDesc,
		&tx.CallbackPayload, &tx.CreatedAt, &tx.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get mpesa transaction: %w", err)
	}

	return &tx, nil
}

func (r *MpesaRepository) UpdateCallback(ctx context.Context, checkoutRequestID string, resultCode int, resultDesc, mpesaReceipt string, payload any) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal callback payload: %w", err)
	}

	status := "failed"
	if resultCode == 0 {
		status = "completed"
	}

	_, err = r.db.Exec(ctx, `
		UPDATE mpesa_transactions
		SET status = $1, result_code = $2, result_desc = $3, mpesa_receipt = $4,
		    callback_payload = $5, updated_at = NOW()
		WHERE checkout_request_id = $6
	`, status, resultCode, resultDesc, mpesaReceipt, payloadJSON, checkoutRequestID)
	if err != nil {
		return fmt.Errorf("failed to update mpesa transaction: %w", err)
	}

	return nil
}

func (r *MpesaRepository) LinkTransaction(ctx context.Context, mpesaID, transactionID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE mpesa_transactions SET transaction_id = $1 WHERE id = $2
	`, transactionID, mpesaID)
	if err != nil {
		return fmt.Errorf("failed to link transaction: %w", err)
	}
	return nil
}
