package handler

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/Rohianon/equishare-global-trading/pkg/errors"
	"github.com/Rohianon/equishare-global-trading/pkg/events"
	"github.com/Rohianon/equishare-global-trading/pkg/logger"
	"github.com/Rohianon/equishare-global-trading/pkg/mpesa"
	"github.com/Rohianon/equishare-global-trading/services/payment-service/internal/repository"
	"github.com/Rohianon/equishare-global-trading/services/payment-service/internal/types"
)

type MpesaClient interface {
	STKPush(ctx context.Context, phone string, amount int, reference string) (*mpesa.STKPushResponse, error)
}

type SMSClient interface {
	Send(to, message string) error
}

type Handler struct {
	userRepo   *repository.UserRepository
	walletRepo *repository.WalletRepository
	mpesaRepo  *repository.MpesaRepository
	mpesa      MpesaClient
	sms        SMSClient
	publisher  events.Publisher
}

func New(
	userRepo *repository.UserRepository,
	walletRepo *repository.WalletRepository,
	mpesaRepo *repository.MpesaRepository,
	mpesa MpesaClient,
	sms SMSClient,
	publisher events.Publisher,
) *Handler {
	return &Handler{
		userRepo:   userRepo,
		walletRepo: walletRepo,
		mpesaRepo:  mpesaRepo,
		mpesa:      mpesa,
		sms:        sms,
		publisher:  publisher,
	}
}

func (h *Handler) Deposit(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req types.DepositRequest
	if err := c.BodyParser(&req); err != nil {
		return apperrors.ErrValidation.WithDetails("Invalid request body")
	}

	if req.Amount < 10 {
		return apperrors.ErrValidation.WithDetails("Minimum deposit is KES 10")
	}
	if req.Amount > 150000 {
		return apperrors.ErrValidation.WithDetails("Maximum deposit is KES 150,000")
	}

	ctx := c.Context()

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get user")
		return apperrors.ErrInternal
	}

	if !user.IsActive {
		return apperrors.ErrForbidden.WithDetails("Account is deactivated")
	}

	wallet, err := h.walletRepo.GetByUserAndCurrency(ctx, userID, "KES")
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get wallet")
		return apperrors.ErrInternal
	}

	reference := fmt.Sprintf("EQS-%s", userID[:8])
	stkResp, err := h.mpesa.STKPush(ctx, user.Phone, req.Amount, reference)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to initiate STK push")
		return apperrors.ErrServiceUnavailable.WithDetails("Failed to initiate M-Pesa payment")
	}

	_, err = h.mpesaRepo.Create(ctx, userID, stkResp.CheckoutRequestID, stkResp.MerchantRequestID, user.Phone, float64(req.Amount))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to save mpesa transaction")
	}

	if h.publisher != nil {
		h.publisher.Publish(ctx, events.TopicPaymentInitiated, events.NewEvent(
			events.EventTypePaymentInitiated,
			"payment-service",
			map[string]any{
				"user_id":             userID,
				"wallet_id":           wallet.ID,
				"amount":              req.Amount,
				"currency":            "KES",
				"checkout_request_id": stkResp.CheckoutRequestID,
			},
		))
	}

	logger.Info().
		Str("user_id", userID).
		Int("amount", req.Amount).
		Str("checkout_request_id", stkResp.CheckoutRequestID).
		Msg("STK push initiated")

	return c.Status(fiber.StatusOK).JSON(types.DepositResponse{
		CheckoutRequestID: stkResp.CheckoutRequestID,
		Message:           "STK Push sent to your phone. Enter your M-Pesa PIN to complete.",
		Amount:            req.Amount,
		Currency:          "KES",
	})
}

func (h *Handler) STKCallback(c *fiber.Ctx) error {
	var callback mpesa.STKCallback
	if err := c.BodyParser(&callback); err != nil {
		logger.Error().Err(err).Msg("Failed to parse STK callback")
		return c.Status(fiber.StatusOK).JSON(types.WebhookResponse{
			ResultCode: 0,
			ResultDesc: "Accepted",
		})
	}

	data := mpesa.ParseCallback(&callback)
	ctx := c.Context()

	logger.Info().
		Str("checkout_request_id", data.CheckoutRequestID).
		Int("result_code", data.ResultCode).
		Str("result_desc", data.ResultDesc).
		Msg("Received STK callback")

	mpesaTx, err := h.mpesaRepo.GetByCheckoutRequestID(ctx, data.CheckoutRequestID)
	if err != nil {
		logger.Error().Err(err).Str("checkout_request_id", data.CheckoutRequestID).Msg("Mpesa transaction not found")
		return c.Status(fiber.StatusOK).JSON(types.WebhookResponse{
			ResultCode: 0,
			ResultDesc: "Accepted",
		})
	}

	if mpesaTx.Status != "pending" {
		logger.Warn().Str("checkout_request_id", data.CheckoutRequestID).Msg("Duplicate callback ignored")
		return c.Status(fiber.StatusOK).JSON(types.WebhookResponse{
			ResultCode: 0,
			ResultDesc: "Accepted",
		})
	}

	err = h.mpesaRepo.UpdateCallback(ctx, data.CheckoutRequestID, data.ResultCode, data.ResultDesc, data.MpesaReceiptNo, callback)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to update mpesa transaction")
	}

	if data.ResultCode == 0 {
		wallet, err := h.walletRepo.GetByUserAndCurrency(ctx, mpesaTx.UserID, "KES")
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get wallet")
		} else {
			err = h.walletRepo.Credit(ctx, wallet.ID, data.Amount)
			if err != nil {
				logger.Error().Err(err).Msg("Failed to credit wallet")
			} else {
				description := fmt.Sprintf("M-Pesa deposit - %s", data.MpesaReceiptNo)
				tx, err := h.walletRepo.CreateTransaction(ctx, mpesaTx.UserID, wallet.ID, "deposit", "mpesa", data.MpesaReceiptNo, data.Amount, description)
				if err != nil {
					logger.Error().Err(err).Msg("Failed to create transaction record")
				} else {
					h.mpesaRepo.LinkTransaction(ctx, mpesaTx.ID, tx.ID)
				}

				if h.publisher != nil {
					h.publisher.Publish(ctx, events.TopicPaymentCompleted, events.NewEvent(
						events.EventTypePaymentCompleted,
						"payment-service",
						map[string]any{
							"user_id":        mpesaTx.UserID,
							"wallet_id":      wallet.ID,
							"amount":         data.Amount,
							"currency":       "KES",
							"mpesa_receipt":  data.MpesaReceiptNo,
							"transaction_id": tx.ID,
						},
					))
				}

				user, _ := h.userRepo.GetByID(ctx, mpesaTx.UserID)
				if user != nil && h.sms != nil {
					msg := fmt.Sprintf("Your EquiShare wallet has been credited with KES %.2f. Receipt: %s. New balance: KES %.2f",
						data.Amount, data.MpesaReceiptNo, wallet.Balance+data.Amount)
					h.sms.Send(user.Phone, msg)
				}

				logger.Info().
					Str("user_id", mpesaTx.UserID).
					Float64("amount", data.Amount).
					Str("mpesa_receipt", data.MpesaReceiptNo).
					Msg("Deposit completed successfully")
			}
		}
	} else {
		if h.publisher != nil {
			h.publisher.Publish(ctx, events.TopicPaymentFailed, events.NewEvent(
				events.EventTypePaymentFailed,
				"payment-service",
				map[string]any{
					"user_id":             mpesaTx.UserID,
					"amount":              mpesaTx.Amount,
					"result_code":         data.ResultCode,
					"result_desc":         data.ResultDesc,
					"checkout_request_id": data.CheckoutRequestID,
				},
			))
		}

		logger.Info().
			Str("user_id", mpesaTx.UserID).
			Int("result_code", data.ResultCode).
			Str("result_desc", data.ResultDesc).
			Msg("Deposit failed")
	}

	return c.Status(fiber.StatusOK).JSON(types.WebhookResponse{
		ResultCode: 0,
		ResultDesc: "Accepted",
	})
}

func (h *Handler) GetWalletBalance(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := c.Context()

	wallet, err := h.walletRepo.GetByUserAndCurrency(ctx, userID, "KES")
	if err != nil {
		return apperrors.ErrNotFound.WithDetails("Wallet not found")
	}

	return c.JSON(fiber.Map{
		"currency":  wallet.Currency,
		"available": wallet.Balance,
		"pending":   wallet.LockedBalance,
		"total":     wallet.Balance + wallet.LockedBalance,
	})
}

func (h *Handler) GetTransactions(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := c.Context()

	page := c.QueryInt("page", 1)
	perPage := c.QueryInt("per_page", 10)

	transactions, total, err := h.walletRepo.GetTransactions(ctx, userID, page, perPage)
	if err != nil {
		logger.Error().Err(err).Str("user_id", userID).Msg("Failed to get transactions")
		return apperrors.ErrInternal
	}

	txResponses := make([]fiber.Map, len(transactions))
	for i, tx := range transactions {
		description := ""
		if tx.Description != nil {
			description = *tx.Description
		}
		txResponses[i] = fiber.Map{
			"id":          tx.ID,
			"type":        tx.Type,
			"amount":      tx.Amount,
			"currency":    tx.Currency,
			"status":      tx.Status,
			"description": description,
			"created_at":  tx.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return c.JSON(fiber.Map{
		"transactions": txResponses,
		"total":        total,
		"page":         page,
		"per_page":     perPage,
	})
}
