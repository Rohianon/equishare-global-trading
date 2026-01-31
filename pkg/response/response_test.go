package response

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	apperrors "github.com/Rohianon/equishare-global-trading/pkg/errors"
)

func TestSuccess(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Success(c, map[string]string{"key": "value"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}

	if result.Error != nil {
		t.Error("error should be nil for success response")
	}
	if result.Meta.RequestID == "" {
		t.Error("request_id should be set")
	}
	if result.Meta.Timestamp.IsZero() {
		t.Error("timestamp should be set")
	}
}

func TestCreated(t *testing.T) {
	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		return Created(c, map[string]string{"id": "123"})
	})

	req := httptest.NewRequest("POST", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("status = %d, want 201", resp.StatusCode)
	}
}

func TestNoContent(t *testing.T) {
	app := fiber.New()
	app.Delete("/test", func(c *fiber.Ctx) error {
		return NoContent(c)
	})

	req := httptest.NewRequest("DELETE", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		t.Errorf("status = %d, want 204", resp.StatusCode)
	}
}

func TestPaginated(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		items := []string{"a", "b", "c"}
		return Paginated(c, items, 1, 20, 100)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data struct {
			Items      []string   `json:"items"`
			Pagination Pagination `json:"pagination"`
		} `json:"data"`
		Meta Meta `json:"meta"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}

	if len(result.Data.Items) != 3 {
		t.Errorf("items length = %d, want 3", len(result.Data.Items))
	}
	if result.Data.Pagination.Page != 1 {
		t.Errorf("page = %d, want 1", result.Data.Pagination.Page)
	}
	if result.Data.Pagination.PerPage != 20 {
		t.Errorf("per_page = %d, want 20", result.Data.Pagination.PerPage)
	}
	if result.Data.Pagination.Total != 100 {
		t.Errorf("total = %d, want 100", result.Data.Pagination.Total)
	}
	if result.Data.Pagination.TotalPages != 5 {
		t.Errorf("total_pages = %d, want 5", result.Data.Pagination.TotalPages)
	}
	if !result.Data.Pagination.HasMore {
		t.Error("has_more should be true")
	}
}

func TestPaginatedLastPage(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		items := []string{"a"}
		return Paginated(c, items, 5, 20, 100)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Data struct {
			Pagination Pagination `json:"pagination"`
		} `json:"data"`
	}
	json.Unmarshal(body, &result)

	if result.Data.Pagination.HasMore {
		t.Error("has_more should be false on last page")
	}
}

func TestError(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Error(c, 400, "VALIDATION_ERROR", "Invalid input", "field 'x' required")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result Response
	json.Unmarshal(body, &result)

	if result.Error == nil {
		t.Fatal("error should not be nil")
	}
	if result.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("error.code = %s, want VALIDATION_ERROR", result.Error.Code)
	}
	if result.Error.Message != "Invalid input" {
		t.Errorf("error.message = %s, want Invalid input", result.Error.Message)
	}
	if len(result.Error.Details) != 1 {
		t.Errorf("error.details length = %d, want 1", len(result.Error.Details))
	}
}

func TestErrorHandler_AppError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})
	app.Get("/test", func(c *fiber.Ctx) error {
		return apperrors.ErrInsufficientFunds.WithDetails("Balance: 100, Required: 500")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result Response
	json.Unmarshal(body, &result)

	if result.Error.Code != "PAYMENT_INSUFFICIENT_FUNDS" {
		t.Errorf("error.code = %s, want PAYMENT_INSUFFICIENT_FUNDS", result.Error.Code)
	}
}

func TestErrorHandler_FiberError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.ErrNotFound
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result Response
	json.Unmarshal(body, &result)

	if result.Error.Code != "NOT_FOUND" {
		t.Errorf("error.code = %s, want NOT_FOUND", result.Error.Code)
	}
}

func TestErrorHandler_GenericError(t *testing.T) {
	app := fiber.New(fiber.Config{
		ErrorHandler: ErrorHandler,
	})
	app.Get("/test", func(c *fiber.Ctx) error {
		return fiber.NewError(500, "something went wrong")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result Response
	json.Unmarshal(body, &result)

	if result.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("error.code = %s, want INTERNAL_ERROR", result.Error.Code)
	}
}

func TestRequestIDFromHeader(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Success(c, nil)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom-request-id")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result Response
	json.Unmarshal(body, &result)

	if result.Meta.RequestID != "custom-request-id" {
		t.Errorf("request_id = %s, want custom-request-id", result.Meta.RequestID)
	}
}

func TestPaginationCalculation(t *testing.T) {
	tests := []struct {
		total      int64
		perPage    int
		wantPages  int
	}{
		{100, 20, 5},
		{101, 20, 6},
		{99, 20, 5},
		{20, 20, 1},
		{0, 20, 0},
		{1, 20, 1},
	}

	for _, tt := range tests {
		totalPages := int(tt.total) / tt.perPage
		if int(tt.total)%tt.perPage > 0 {
			totalPages++
		}
		if totalPages != tt.wantPages {
			t.Errorf("total=%d, perPage=%d: got %d pages, want %d", tt.total, tt.perPage, totalPages, tt.wantPages)
		}
	}
}
