package api

import (
	"fmt"
	"strings"
)

// NMIError represents a structured error response from NMI
type NMIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Raw     string `json:"raw,omitempty"`
}

func (e *NMIError) Error() string {
	return fmt.Sprintf("NMI Error %s: %s", e.Code, e.Message)
}

// Common error codes
const (
	ErrInvalidCard          = "invalid_card"
	ErrInvalidAmount        = "invalid_amount"
	ErrInvalidRequest       = "invalid_request"
	ErrDuplicateTransaction = "duplicate_transaction"
	ErrProcessingError      = "processing_error"
	ErrPartialResponse      = "partial_response"
	ErrInvalidRefund        = "invalid_refund"
	ErrNetworkError         = "network_error"
	ErrAuthenticationFailed = "authentication_failed"
	ErrInvalidAction        = "invalid_action"
	ErrSystemError          = "system_error"
)

// NewNMIError creates a new NMIError
func NewNMIError(code, message string, rawResponse string) *NMIError {
	return &NMIError{
		Code:    code,
		Message: message,
		Raw:     rawResponse,
	}
}

// ParseNMIErrorResponse parses NMI's error response
func ParseNMIErrorResponse(responseText, responseCode, rawResponse string) *NMIError {
	// Map NMI response codes to our error codes
	codeMap := map[string]string{
		"200": ErrInvalidCard,
		"201": ErrInvalidAmount,
		"300": ErrAuthenticationFailed,
		"400": ErrProcessingError,
		"500": ErrSystemError,
		"600": ErrInvalidAction,
		"601": ErrDuplicateTransaction,
		"700": ErrNetworkError,
	}

	// Extract more detailed error information if available
	details := ""
	if strings.Contains(responseText, "REFID:") {
		details = responseText[strings.Index(responseText, "REFID:"):]
	}

	code, exists := codeMap[responseCode]
	if !exists {
		code = ErrProcessingError
	}

	return &NMIError{
		Code:    code,
		Message: responseText,
		Details: details,
		Raw:     rawResponse,
	}
}
