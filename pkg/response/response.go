package response

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// =============================================================================
// Standard API Response Envelope
// =============================================================================
// All API responses MUST use this envelope format for consistency across services.
//
// Success Response:
//
//	{
//	  "data": { ... },
//	  "meta": {
//	    "request_id": "uuid",
//	    "timestamp": "2026-01-31T12:00:00Z"
//	  }
//	}
//
// Error Response:
//
//	{
//	  "error": {
//	    "code": "VALIDATION_ERROR",
//	    "message": "Invalid input",
//	    "details": ["field 'email' is required"]
//	  },
//	  "meta": {
//	    "request_id": "uuid",
//	    "timestamp": "2026-01-31T12:00:00Z"
//	  }
//	}
// =============================================================================

// Response is the standard API response envelope
type Response struct {
	Data  any        `json:"data,omitempty"`
	Error *ErrorBody `json:"error,omitempty"`
	Meta  Meta       `json:"meta"`
}

// ErrorBody contains error details
type ErrorBody struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

// Meta contains request metadata
type Meta struct {
	RequestID string    `json:"request_id"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version,omitempty"`
}

// PaginatedData wraps paginated results
type PaginatedData struct {
	Items      any        `json:"items"`
	Pagination Pagination `json:"pagination"`
}

// Pagination contains pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasMore    bool  `json:"has_more"`
}

// =============================================================================
// Response Builders
// =============================================================================

// Success returns a successful response with data
func Success(c *fiber.Ctx, data any) error {
	return c.JSON(Response{
		Data: data,
		Meta: buildMeta(c),
	})
}

// SuccessWithStatus returns a successful response with custom status code
func SuccessWithStatus(c *fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(Response{
		Data: data,
		Meta: buildMeta(c),
	})
}

// Created returns a 201 Created response
func Created(c *fiber.Ctx, data any) error {
	return SuccessWithStatus(c, fiber.StatusCreated, data)
}

// NoContent returns a 204 No Content response
func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// Paginated returns a paginated response
func Paginated(c *fiber.Ctx, items any, page, perPage int, total int64) error {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}

	return c.JSON(Response{
		Data: PaginatedData{
			Items: items,
			Pagination: Pagination{
				Page:       page,
				PerPage:    perPage,
				Total:      total,
				TotalPages: totalPages,
				HasMore:    page < totalPages,
			},
		},
		Meta: buildMeta(c),
	})
}

// Error returns an error response
func Error(c *fiber.Ctx, status int, code, message string, details ...string) error {
	return c.Status(status).JSON(Response{
		Error: &ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: buildMeta(c),
	})
}

// =============================================================================
// Helpers
// =============================================================================

func buildMeta(c *fiber.Ctx) Meta {
	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		requestID = c.Locals("request_id", uuid.New().String()).(string)
	}

	return Meta{
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
		Version:   "v1",
	}
}

// GetRequestID extracts request ID from context
func GetRequestID(c *fiber.Ctx) string {
	if id := c.Get("X-Request-ID"); id != "" {
		return id
	}
	if id, ok := c.Locals("request_id").(string); ok {
		return id
	}
	return ""
}
