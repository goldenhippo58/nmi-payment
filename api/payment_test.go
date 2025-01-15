package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePaymentRequest(t *testing.T) {
    tests := []struct {
        name    string
        req     PaymentRequest
        wantErr bool
        errCode string
    }{
        {
            name: "Valid Request",
            req: PaymentRequest{
                Amount:     "10.99",
                CreditCard: "4111111111111111",
                ExpDate:    "1225",
                CVV:       "123",
                Type:      "sale",
            },
            wantErr: false,
        },
        {
            name: "Invalid Amount Format",
            req: PaymentRequest{
                Amount:     "10.9",
                CreditCard: "4111111111111111",
                ExpDate:    "1225",
                CVV:       "123",
                Type:      "sale",
            },
            wantErr: true,
            errCode: ErrInvalidAmount,
        },
        {
            name: "Expired Card",
            req: PaymentRequest{
                Amount:     "10.99",
                CreditCard: "4111111111111111",
                ExpDate:    "0123", // January 2023
                CVV:       "123",
                Type:      "sale",
            },
            wantErr: true,
            errCode: ErrInvalidCard,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePaymentRequest(tt.req)
            if tt.wantErr {
                assert.Error(t, err)
                if nmiErr, ok := err.(*NMIError); ok {
                    assert.Equal(t, tt.errCode, nmiErr.Code)
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}