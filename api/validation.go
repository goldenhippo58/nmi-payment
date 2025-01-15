package api

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ValidatePaymentRequest validates payment request parameters
func ValidatePaymentRequest(req PaymentRequest) error {
	// Validate amount
	if req.Amount == "" {
		return fmt.Errorf("amount is required")
	}
	if err := validateAmount(req.Amount); err != nil {
		return err
	}

	// Validate type
	if req.Type == "" {
		return fmt.Errorf("type is required")
	}
	if err := validateTransactionType(req.Type); err != nil {
		return err
	}

	// If not using customer vault, validate card details
	if req.CustomerVaultID == "" {
		if req.CreditCard == "" || req.ExpDate == "" || req.CVV == "" {
			return fmt.Errorf("either customer_vault_id or credit_card, exp_date, and cvv are required")
		}

		// Validate each card detail
		if err := validateCreditCard(req.CreditCard); err != nil {
			return err
		}
		if err := validateExpirationDate(req.ExpDate); err != nil {
			return err
		}
		if err := validateCVV(req.CVV); err != nil {
			return err
		}
	} else {
		// Validate customer vault ID
		if len(req.CustomerVaultID) < 8 {
			return fmt.Errorf("customer_vault_id must be at least 8 characters")
		}
	}

	// Validate billing info if provided
	if req.Billing != nil {
		if err := validateBillingInfo(req.Billing); err != nil {
			return err
		}
	}

	return nil
}

// ValidateRefundRequest validates refund request parameters
func ValidateRefundRequest(req RefundRequest, originalAmount string) error {
	if req.TransactionID == "" {
		return NewNMIError(ErrInvalidRequest, "transaction_id is required", "")
	}

	if req.Amount != "" {
		// Validate amount format
		amountRegex := regexp.MustCompile(`^\d+\.\d{2}$`)
		if !amountRegex.MatchString(req.Amount) {
			return NewNMIError(ErrInvalidAmount, "invalid amount format: must be in dollars.cents format (e.g., 10.99)", "")
		}

		// Validate refund amount is greater than 0
		refundAmount, _ := strconv.ParseFloat(req.Amount, 64)
		if refundAmount <= 0 {
			return NewNMIError(ErrInvalidRefund, "refund amount must be greater than 0", "")
		}

		// Check if refund amount exceeds original amount
		if originalAmount != "" {
			originalAmt, _ := strconv.ParseFloat(originalAmount, 64)
			if refundAmount > originalAmt {
				return NewNMIError(ErrInvalidRefund, "refund amount cannot exceed original transaction amount", "")
			}
		}
	}

	return nil
}

// ValidateRecurringRequest validates recurring payment request parameters
func ValidateRecurringRequest(req RecurringPaymentRequest) error {
	if req.CustomerVaultID == "" {
		return NewNMIError(ErrInvalidRequest, "customer_vault_id is required", "")
	}
	if req.PlanID == "" {
		return NewNMIError(ErrInvalidRequest, "plan_id is required", "")
	}
	if req.Amount == "" {
		return NewNMIError(ErrInvalidRequest, "amount is required", "")
	}
	if req.BillingCycle == "" {
		return NewNMIError(ErrInvalidRequest, "billing_cycle is required", "")
	}
	if err := validateAmount(req.Amount); err != nil {
		return err
	}
	if err := validateBillingCycle(req.BillingCycle); err != nil {
		return err
	}
	if req.StartDate != "" {
		if err := validateStartDate(req.StartDate); err != nil {
			return err
		}
	}
	if req.Billing != nil {
		if err := validateBillingInfo(req.Billing); err != nil {
			return err
		}
	}
	return nil
}

// Helper function to validate start_date
func validateStartDate(startDate string) error {
	if _, err := time.Parse("01/02/2006", startDate); err != nil {
		return NewNMIError(ErrInvalidRequest, "invalid start_date format (expected MM/DD/YYYY)", "")
	}
	return nil
}

// Helper validation functions
func validateAmount(amount string) error {
	amountRegex := regexp.MustCompile(`^\d+\.\d{2}$`)
	if !amountRegex.MatchString(amount) {
		return NewNMIError(ErrInvalidAmount, "invalid amount format: must be in dollars.cents format (e.g., 10.99)", "")
	}

	amountFloat, _ := strconv.ParseFloat(amount, 64)
	if amountFloat <= 0 {
		return NewNMIError(ErrInvalidAmount, "amount must be greater than zero", "")
	}

	return nil
}

func validateCreditCard(number string) error {
	// Remove any spaces or hyphens
	number = regexp.MustCompile(`[\s-]`).ReplaceAllString(number, "")

	if !regexp.MustCompile(`^\d{13,19}$`).MatchString(number) {
		return NewNMIError(ErrInvalidCard, "invalid credit card number length", "")
	}

	// Luhn algorithm check
	sum := 0
	alternate := false
	for i := len(number) - 1; i >= 0; i-- {
		n, _ := strconv.Atoi(string(number[i]))
		if alternate {
			n *= 2
			if n > 9 {
				n = (n % 10) + 1
			}
		}
		sum += n
		alternate = !alternate
	}

	if sum%10 != 0 {
		return NewNMIError(ErrInvalidCard, "invalid credit card number (failed Luhn check)", "")
	}

	return nil
}

func validateExpirationDate(expDate string) error {
	if !regexp.MustCompile(`^(0[1-9]|1[0-2])\d{2}$`).MatchString(expDate) {
		return NewNMIError(ErrInvalidCard, "invalid expiration date format (must be MMYY)", "")
	}

	month, _ := strconv.Atoi(expDate[:2])
	year, _ := strconv.Atoi(expDate[2:])

	now := time.Now()
	currentYear := now.Year() % 100
	currentMonth := int(now.Month())

	if year < currentYear || (year == currentYear && month < currentMonth) {
		return NewNMIError(ErrInvalidCard, "card has expired", "")
	}

	return nil
}

func validateCVV(cvv string) error {
	if !regexp.MustCompile(`^\d{3,4}$`).MatchString(cvv) {
		return NewNMIError(ErrInvalidCard, "invalid CVV (must be 3 or 4 digits)", "")
	}
	return nil
}

func validateTransactionType(txType string) error {
	validTypes := map[string]bool{
		"sale":     true,
		"auth":     true,
		"capture":  true,
		"credit":   true,
		"validate": true,
		"void":     true,
		"refund":   true,
	}

	if !validTypes[strings.ToLower(txType)] {
		return NewNMIError(ErrInvalidRequest, "invalid transaction type", "")
	}

	return nil
}

func validateBillingInfo(billing *BillingInfo) error {
	if billing.FirstName == "" || billing.LastName == "" {
		return NewNMIError(ErrInvalidRequest, "first_name and last_name are required", "")
	}

	if billing.Address1 == "" || billing.City == "" || billing.State == "" || billing.Zip == "" {
		return NewNMIError(ErrInvalidRequest, "complete address is required", "")
	}

	// Validate email if provided
	if billing.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(billing.Email) {
			return NewNMIError(ErrInvalidRequest, "invalid email format", "")
		}
	}

	// Validate phone if provided
	if billing.Phone != "" {
		phone := regexp.MustCompile(`\D`).ReplaceAllString(billing.Phone, "")
		if len(phone) < 10 {
			return NewNMIError(ErrInvalidRequest, "invalid phone number", "")
		}
	}

	return nil
}

func validateBillingCycle(cycle string) error {
	validCycles := map[string]bool{
		"daily":     true,
		"weekly":    true,
		"monthly":   true,
		"quarterly": true,
		"yearly":    true,
	}

	if !validCycles[strings.ToLower(cycle)] {
		return NewNMIError(ErrInvalidRequest, "invalid billing cycle", "")
	}

	return nil
}
