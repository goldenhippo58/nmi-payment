package tests

import (
	"context"
	"testing"

	"nmi-pay-int/api"

	"github.com/stretchr/testify/assert"
)

func TestPaymentIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration tests")
	}

	ctx := context.Background()

	// Test Sale Transaction
	t.Run("Process Sale", func(t *testing.T) {
		req := api.PaymentRequest{
			Amount:     "10.99",
			CreditCard: "4111111111111111",
			ExpDate:    "1225",
			CVV:        "123",
			Type:       "sale",
		}

		resp, err := api.ProcessPayment(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "1", resp.Response)
		assert.NotEmpty(t, resp.TransactionID)
	})

	// Test Tokenization
	t.Run("Process Tokenization", func(t *testing.T) {
		req := api.PaymentRequest{
			CreditCard: "4111111111111111",
			ExpDate:    "1225",
			CVV:        "123",
		}

		resp, err := api.ProcessTokenization(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.CustomerVaultID)
	})
}
