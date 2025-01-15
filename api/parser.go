package api

import (
	"fmt"
	"net/url"
)

// NMIResponse represents the common response structure
type NMIResponse struct {
	Response        string `json:"response"`
	ResponseText    string `json:"responsetext"`
	AuthCode        string `json:"authcode"`
	TransactionID   string `json:"transactionid"`
	AVSResponse     string `json:"avsresponse"`
	CVVResponse     string `json:"cvvresponse"`
	OrderID         string `json:"orderid"`
	Type            string `json:"type"`
	ResponseCode    string `json:"response_code"`
	Amount          string `json:"amount,omitempty"`
	CustomerVaultID string `json:"customer_vault_id,omitempty"`
}

// ParseNMIResponse parses NMI's response with enhanced error handling
func ParseNMIResponse(rawResponse string) (*NMIResponse, error) {
	values, err := url.ParseQuery(rawResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NMI response: %w", err)
	}

	response := &NMIResponse{
		Response:        values.Get("response"),
		ResponseText:    values.Get("responsetext"),
		AuthCode:        values.Get("authcode"),
		TransactionID:   values.Get("transactionid"),
		AVSResponse:     values.Get("avsresponse"),
		CVVResponse:     values.Get("cvvresponse"),
		OrderID:         values.Get("orderid"),
		Type:            values.Get("type"),
		ResponseCode:    values.Get("response_code"),
		Amount:          values.Get("amount"),
		CustomerVaultID: values.Get("customer_vault_id"),
	}

	if response.Response != "1" {
		return response, ParseNMIErrorResponse(response.ResponseText, response.ResponseCode, rawResponse)
	}

	return response, nil
}

// AVS Response Codes
const (
	AVSExactMatchZIP9     = "X" // Exact match, 9-character numeric ZIP
	AVSExactMatchZIP5     = "Y" // Exact match, 5-character numeric ZIP
	AVSExactMatchAddr     = "A" // Address match only
	AVSNoMatch            = "N" // No address or ZIP match
	AVSAddressMatchOnly   = "B" // Address match only
	AVSZIPMatchOnly       = "P" // ZIP match only
	AVSUnavailable        = "U" // Address information unavailable
	AVSNotSupported       = "S" // Service not supported
	AVSRetry              = "R" // Retry, system unavailable
	AVSError              = "E" // Error in transaction data
	AVSInternationalUnsup = "G" // Non-U.S. issuer does not participate
)

// CVV Response Codes
const (
	CVVMatch        = "M" // CVV2/CVC2 match
	CVVNoMatch      = "N" // CVV2/CVC2 no match
	CVVNotProcessed = "P" // Not processed
	CVVNoSubmit     = "S" // Merchant has indicated CVV not present
	CVVUnavailable  = "U" // Issuer not certified
)

// Helper function to check AVS response
func IsAVSMatch(avsResponse string) bool {
	return avsResponse == AVSExactMatchZIP9 || avsResponse == AVSExactMatchZIP5
}

// Helper function to check CVV response
func IsCVVMatch(cvvResponse string) bool {
	return cvvResponse == CVVMatch
}

// Helper function to extract specific value from NMI response
func ExtractValue(rawResponse, key string) string {
	values, err := url.ParseQuery(rawResponse)
	if err != nil {
		return ""
	}
	return values.Get(key)
}
