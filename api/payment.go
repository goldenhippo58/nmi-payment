package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"nmi-pay-int/metrics"

	"github.com/gorilla/mux"
)

var (
	transactionCache = struct {
		sync.RWMutex
		processed map[string]bool
	}{processed: make(map[string]bool)}
)

// Request Structures
type PaymentRequest struct {
	APIKey           string       `json:"api_key,omitempty"`
	Amount           string       `json:"amount"`
	CreditCard       string       `json:"credit_card,omitempty"`
	ExpDate          string       `json:"exp_date,omitempty"`
	CVV              string       `json:"cvv,omitempty"`
	Token            string       `json:"token,omitempty"`
	CustomerVaultID  string       `json:"customer_vault_id,omitempty"`
	Type             string       `json:"type"`
	OrderID          string       `json:"order_id,omitempty"`
	CustomerID       string       `json:"customer_id,omitempty"`
	IdempotencyKey   string       `json:"idempotency_key,omitempty"`
	RecurringPayment bool         `json:"recurring_payment,omitempty"`
	PlanID           string       `json:"plan_id,omitempty"`
	Billing          *BillingInfo `json:"billing,omitempty"`
}

type BillingInfo struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Address1  string `json:"address1"`
	City      string `json:"city"`
	State     string `json:"state"`
	Zip       string `json:"zip"`
	Country   string `json:"country"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
}

// Response Structures
type PaymentResponse struct {
	RawResponse     string `json:"raw_response"`
	StatusCode      int    `json:"status_code"`
	Response        string `json:"response"`
	ResponseText    string `json:"responsetext"`
	AuthCode        string `json:"authcode"`
	TransactionID   string `json:"transactionid"`
	AVSResponse     string `json:"avsresponse"`
	CVVResponse     string `json:"cvvresponse"`
	OrderID         string `json:"orderid"`
	Type            string `json:"type"`
	ResponseCode    string `json:"response_code"`
	ErrorMessage    string `json:"error_message,omitempty"`
	CustomerVaultID string `json:"customer_vault_id,omitempty"`
}

type RefundResponse struct {
	RawResponse   string `json:"raw_response"`
	StatusCode    int    `json:"status_code"`
	Response      string `json:"response"`
	ResponseText  string `json:"responsetext"`
	AuthCode      string `json:"authcode"`
	TransactionID string `json:"transactionid"`
	Type          string `json:"type"`
	ResponseCode  string `json:"response_code"`
	Amount        string `json:"amount"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

type VoidResponse struct {
	RawResponse   string `json:"raw_response"`
	StatusCode    int    `json:"status_code"`
	Response      string `json:"response"`
	ResponseText  string `json:"responsetext"`
	AuthCode      string `json:"authcode"`
	TransactionID string `json:"transactionid"`
	Type          string `json:"type"`
	ResponseCode  string `json:"response_code"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

type LookupResponse struct {
	RawResponse   string `json:"raw_response"`
	StatusCode    int    `json:"status_code"`
	Response      string `json:"response"`
	ResponseText  string `json:"responsetext"`
	TransactionID string `json:"transactionid"`
	Type          string `json:"type"`
	Amount        string `json:"amount"`
	ResponseCode  string `json:"response_code"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

type TokenizeResponse struct {
	CustomerVaultID string `json:"customer_vault_id"`
	Token           string `json:"token"`
	Masked          string `json:"masked_card"`
	CardType        string `json:"card_type"`
	ExpiryDate      string `json:"expiry_date"`
	Success         bool   `json:"success"`
	Message         string `json:"message"`
}

type RecurringResponse struct {
	SubscriptionID  string `json:"subscription_id"`
	Status          string `json:"status"`
	NextBilling     string `json:"next_billing_date"`
	PlanID          string `json:"plan_id"`
	Amount          string `json:"amount"`
	CustomerVaultID string `json:"customer_vault_id"`
}

type RefundRequest struct {
	APIKey        string `json:"api_key,omitempty"`
	TransactionID string `json:"transaction_id"`
	Amount        string `json:"amount,omitempty"`
}

type VoidRequest struct {
	APIKey        string `json:"api_key,omitempty"`
	TransactionID string `json:"transaction_id"`
}

type LookupRequest struct {
	APIKey          string `json:"api_key,omitempty"`
	TransactionID   string `json:"transaction_id"`
	Condition       string `json:"condition,omitempty"`
	TransactionType string `json:"transaction_type,omitempty"`
	ActionType      string `json:"action_type,omitempty"`
}

type RecurringPaymentRequest struct {
	APIKey          string       `json:"api_key,omitempty"`
	CustomerVaultID string       `json:"customer_vault_id"`
	PlanID          string       `json:"plan_id"`
	Amount          string       `json:"amount"`
	BillingCycle    string       `json:"billing_cycle"` // monthly, yearly, etc.
	StartDate       string       `json:"start_date,omitempty"`
	Billing         *BillingInfo `json:"billing,omitempty"`
}

type Plan struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Amount         string `json:"amount"`
	DayFrequency   string `json:"day_frequency,omitempty"`
	Payments       string `json:"payments,omitempty"`
	MonthFrequency string `json:"month_frequency,omitempty"`
	DayOfMonth     string `json:"day_of_month,omitempty"`
}

type PlanResponse struct {
	Plan    Plan   `json:"plan"`
	Message string `json:"message"`
}

// ProcessPayment handles all payment transactions
func ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	// Track transaction processing time
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.RecordTransactionMetrics(req.Type, "processed", duration)
	}()

	// Check for duplicate transactions
	if req.IdempotencyKey != "" {
		transactionCache.RLock()
		if _, exists := transactionCache.processed[req.IdempotencyKey]; exists {
			transactionCache.RUnlock()
			return nil, NewNMIError(ErrDuplicateTransaction, "duplicate transaction detected", "")
		}
		transactionCache.RUnlock()
	}

	// Validate the payment request
	if err := ValidatePaymentRequest(req); err != nil {
		metrics.RecordErrorMetrics(req.Type, "validation_error")
		return nil, err
	}

	// Prepare form data for NMI API request
	formData := url.Values{}
	formData.Set("security_key", req.APIKey)
	formData.Set("amount", req.Amount)
	formData.Set("type", req.Type)

	if req.OrderID != "" {
		formData.Set("orderid", req.OrderID)
	}

	// Handle tokenized or vault transactions
	if req.CustomerVaultID != "" {
		formData.Set("customer_vault_id", req.CustomerVaultID)
		metrics.LogDebug(fmt.Sprintf("Using customer vault ID: %s", req.CustomerVaultID))
	} else {
		formData.Set("ccnumber", req.CreditCard)
		formData.Set("ccexp", req.ExpDate)
		formData.Set("cvv", req.CVV)
	}

	// Send the request to NMI
	resp, err := sendRequest(ctx, formData)
	if err != nil {
		metrics.RecordErrorMetrics(req.Type, "network_error")
		return nil, err
	}

	// Parse the NMI response
	parsedResp, err := ParseNMIResponse(resp)
	if err != nil {
		metrics.RecordErrorMetrics(req.Type, "parse_error")
		return nil, err
	}

	// Record idempotency if applicable
	if req.IdempotencyKey != "" {
		transactionCache.Lock()
		transactionCache.processed[req.IdempotencyKey] = true
		transactionCache.Unlock()
	}

	// Return the successful payment response
	return &PaymentResponse{
		RawResponse:     resp,
		StatusCode:      200,
		Response:        parsedResp.Response,
		ResponseText:    parsedResp.ResponseText,
		AuthCode:        parsedResp.AuthCode,
		TransactionID:   parsedResp.TransactionID,
		AVSResponse:     parsedResp.AVSResponse,
		CVVResponse:     parsedResp.CVVResponse,
		OrderID:         parsedResp.OrderID,
		Type:            parsedResp.Type,
		ResponseCode:    parsedResp.ResponseCode,
		CustomerVaultID: req.CustomerVaultID,
	}, nil
}

// ProcessTokenization handles tokenization of card details
func ProcessTokenization(ctx context.Context, req PaymentRequest) (*TokenizeResponse, error) {
	if err := ValidatePaymentRequest(req); err != nil {
		return nil, err
	}

	formData := url.Values{}
	formData.Set("security_key", req.APIKey)
	formData.Set("ccnumber", req.CreditCard)
	formData.Set("ccexp", req.ExpDate)
	formData.Set("cvv", req.CVV)
	formData.Set("amount", "1.00") // Dummy amount for tokenization
	formData.Set("type", "sale")
	formData.Set("customer_vault", "add_customer")

	vaultID := generateUniqueVaultID()
	formData.Set("customer_vault_id", vaultID)

	if req.Billing != nil {
		addBillingInfo(formData, req.Billing)
	}

	resp, err := sendRequest(ctx, formData)
	if err != nil {
		return nil, err
	}

	parsedResp, err := ParseNMIResponse(resp)
	if err != nil {
		return nil, err
	}

	return &TokenizeResponse{
		CustomerVaultID: vaultID,
		Token:           vaultID,
		Masked:          ExtractValue(resp, "cc_number"),
		CardType:        ExtractValue(resp, "card_type"),
		ExpiryDate:      req.ExpDate,
		Success:         parsedResp.Response == "1",
		Message:         parsedResp.ResponseText,
	}, nil
}

// ProcessRefund handles refund transactions
func ProcessRefund(ctx context.Context, req RefundRequest) (*RefundResponse, error) {
	lookupReq := LookupRequest{
		APIKey:        req.APIKey,
		TransactionID: req.TransactionID,
	}

	lookupResp, err := LookupTransaction(ctx, lookupReq)
	if err != nil {
		return nil, err
	}

	if err := ValidateRefundRequest(req, lookupResp.Amount); err != nil {
		return nil, err
	}

	formData := url.Values{}
	formData.Set("security_key", req.APIKey)
	formData.Set("type", "refund")
	formData.Set("transactionid", req.TransactionID)

	if req.Amount != "" {
		formData.Set("amount", req.Amount)
	}

	resp, err := sendRequest(ctx, formData)
	if err != nil {
		return nil, err
	}

	parsedResp, err := ParseNMIResponse(resp)
	if err != nil {
		return nil, err
	}

	return &RefundResponse{
		RawResponse:   resp,
		StatusCode:    200,
		Response:      parsedResp.Response,
		ResponseText:  parsedResp.ResponseText,
		AuthCode:      parsedResp.AuthCode,
		TransactionID: parsedResp.TransactionID,
		Type:          parsedResp.Type,
		ResponseCode:  parsedResp.ResponseCode,
		Amount:        req.Amount,
	}, nil
}

// VoidTransaction handles void transactions
func VoidTransaction(ctx context.Context, req VoidRequest) (*VoidResponse, error) {
	if req.TransactionID == "" {
		return nil, NewNMIError(ErrInvalidRequest, "transaction_id is required", "")
	}

	formData := url.Values{}
	formData.Set("security_key", req.APIKey)
	formData.Set("type", "void")
	formData.Set("transactionid", req.TransactionID)

	resp, err := sendRequest(ctx, formData)
	if err != nil {
		return nil, err
	}

	parsedResp, err := ParseNMIResponse(resp)
	if err != nil {
		return nil, err
	}

	return &VoidResponse{
		RawResponse:   resp,
		StatusCode:    200,
		Response:      parsedResp.Response,
		ResponseText:  parsedResp.ResponseText,
		AuthCode:      parsedResp.AuthCode,
		TransactionID: parsedResp.TransactionID,
		Type:          parsedResp.Type,
		ResponseCode:  parsedResp.ResponseCode,
	}, nil
}

// LookupTransaction retrieves transaction details
func LookupTransaction(ctx context.Context, req LookupRequest) (*LookupResponse, error) {
	if req.TransactionID == "" {
		return nil, NewNMIError(ErrInvalidRequest, "transaction_id is required", "")
	}

	formData := url.Values{}
	formData.Set("security_key", req.APIKey)
	formData.Set("transaction_id", req.TransactionID)
	formData.Set("condition", "complete")
	formData.Set("transaction_type", "cc")
	formData.Set("action_type", "sale")

	fmt.Printf("Lookup Form Data: %s\n", formData.Encode())

	resp, err := sendRequest(ctx, formData)
	if err != nil {
		return nil, err
	}

	parsedResp, err := ParseNMIResponse(resp)
	if err != nil {
		return nil, err
	}

	return &LookupResponse{
		RawResponse:   resp,
		StatusCode:    200,
		Response:      parsedResp.Response,
		ResponseText:  parsedResp.ResponseText,
		TransactionID: parsedResp.TransactionID,
		Type:          parsedResp.Type,
		Amount:        ExtractValue(resp, "amount"),
		ResponseCode:  parsedResp.ResponseCode,
	}, nil
}

// ProcessRecurringPayment sets up recurring payments
func ProcessRecurringPayment(ctx context.Context, req RecurringPaymentRequest) (*RecurringResponse, error) {
	// Log the incoming request
	fmt.Printf("Incoming Recurring Payment Request: %+v\n", req)

	// Ensure the plan_id exists in PlanStore
	PlanStore.RLock()
	defer PlanStore.RUnlock()
	fmt.Printf("PlanStore Contents: %+v\n", PlanStore.Data)

	plan, exists := PlanStore.Data[req.PlanID]
	if !exists {
		fmt.Printf("Plan ID not found: %s\n", req.PlanID)
		return nil, NewNMIError(ErrInvalidRequest, "plan_id does not exist", "")
	}

	// Log the retrieved plan
	fmt.Printf("Retrieved Plan: %+v\n", plan)

	// Prepare form data
	formData := url.Values{}
	formData.Set("security_key", req.APIKey)
	formData.Set("customer_vault_id", req.CustomerVaultID)
	formData.Set("plan_id", plan.ID)
	formData.Set("recurring", "add_subscription")

	// Add billing details
	if req.Billing != nil {
		addBillingInfo(formData, req.Billing)
	}

	// Log outgoing form data
	fmt.Printf("Outgoing Form Data: %s\n", formData.Encode())

	// Send request
	resp, err := sendRequest(ctx, formData)
	if err != nil {
		return nil, err
	}

	parsedResp, err := ParseNMIResponse(resp)
	if err != nil {
		return nil, err
	}

	return &RecurringResponse{
		SubscriptionID:  parsedResp.TransactionID,
		Status:          parsedResp.Response,
		NextBilling:     ExtractValue(resp, "next_billing_date"),
		PlanID:          req.PlanID,
		Amount:          req.Amount,
		CustomerVaultID: req.CustomerVaultID,
	}, nil
}

// UpdateRecurringPayment modifies an existing subscription
func UpdateRecurringPayment(ctx context.Context, req RecurringPaymentRequest, subscriptionID string) (*RecurringResponse, error) {
	if subscriptionID == "" {
		return nil, NewNMIError(ErrInvalidRequest, "subscription_id is required", "")
	}

	formData := url.Values{}
	formData.Set("security_key", req.APIKey)
	formData.Set("subscription_id", subscriptionID)
	formData.Set("recurring", "update_subscription")

	if req.Amount != "" {
		formData.Set("amount", req.Amount)
	}

	if req.BillingCycle != "" {
		formData.Set("billing_cycle", req.BillingCycle)
	}

	// Add updated plan details if available
	if req.PlanID != "" {
		formData.Set("plan_id", req.PlanID)
	}

	// Add updated billing address details
	if req.Billing != nil {
		addBillingInfo(formData, req.Billing)
	}

	resp, err := sendRequest(ctx, formData)
	if err != nil {
		return nil, err
	}

	parsedResp, err := ParseNMIResponse(resp)
	if err != nil {
		return nil, err
	}

	return &RecurringResponse{
		SubscriptionID:  subscriptionID,
		Status:          parsedResp.Response,
		NextBilling:     ExtractValue(resp, "next_billing_date"),
		PlanID:          req.PlanID,
		Amount:          req.Amount,
		CustomerVaultID: req.CustomerVaultID,
	}, nil
}

// CancelRecurringPayment cancels an existing subscription
func CancelRecurringPayment(ctx context.Context, apiKey, subscriptionID string) error {
	if subscriptionID == "" {
		return NewNMIError(ErrInvalidRequest, "subscription_id is required", "")
	}

	formData := url.Values{}
	formData.Set("security_key", apiKey)
	formData.Set("subscription_id", subscriptionID)
	formData.Set("recurring", "delete_subscription")

	resp, err := sendRequest(ctx, formData)
	if err != nil {
		return err
	}

	parsedResp, err := ParseNMIResponse(resp)
	if err != nil {
		return err
	}

	if parsedResp.Response != "1" {
		return ParseNMIErrorResponse(parsedResp.ResponseText, parsedResp.ResponseCode, resp)
	}

	return nil
}

var PlanStore = struct {
	sync.RWMutex
	Data map[string]Plan
}{Data: make(map[string]Plan)}

// HandleAddPlan Adds a new plan
type AddPlanRequest struct {
	EventID   string `json:"event_id"`
	EventType string `json:"event_type"`
	EventBody struct {
		Merchant struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"merchant"`
		Features struct {
			IsTestMode bool `json:"is_test_mode"`
		} `json:"features"`
		Plan Plan `json:"plan"`
	} `json:"event_body"`
}

func HandleAddPlan() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AddPlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// Extract the plan details from the event body
		plan := req.EventBody.Plan
		if plan.ID == "" || plan.Name == "" || plan.Amount == "" {
			http.Error(w, "Plan ID, Name, and Amount are required", http.StatusBadRequest)
			return
		}

		// Check for duplicate plan ID
		PlanStore.RLock()
		_, exists := PlanStore.Data[plan.ID]
		PlanStore.RUnlock()
		if exists {
			http.Error(w, "Plan ID already exists", http.StatusConflict)
			return
		}

		// Add the plan to the PlanStore
		PlanStore.Lock()
		defer PlanStore.Unlock()
		PlanStore.Data[plan.ID] = plan

		// Log the incoming request and PlanStore state
		fmt.Printf("Plan Added: %+v\n", plan)
		fmt.Printf("PlanStore Data: %+v\n", PlanStore.Data)

		// Respond with success
		response := PlanResponse{
			Plan:    plan,
			Message: "Plan added successfully",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			fmt.Printf("Error encoding response: %v\n", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

// HandleUpdatePlan Updates current plan
func HandleUpdatePlan() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var plan Plan
		if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		PlanStore.Lock()
		defer PlanStore.Unlock()

		existingPlan, exists := PlanStore.Data[plan.ID]
		if !exists {
			http.Error(w, "Plan not found", http.StatusNotFound)
			return
		}

		if plan.Name != "" {
			existingPlan.Name = plan.Name
		}
		if plan.Amount != "" {
			existingPlan.Amount = plan.Amount
		}

		PlanStore.Data[plan.ID] = existingPlan

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PlanResponse{
			Plan:    existingPlan,
			Message: "Plan updated successfully",
		})
	}
}

// HandleCancelPlan Cancels current plan
func HandleCancelPlan() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		planID := vars["id"]

		PlanStore.Lock()
		defer PlanStore.Unlock()

		if _, exists := PlanStore.Data[planID]; !exists {
			http.Error(w, "Plan not found", http.StatusNotFound)
			return
		}

		delete(PlanStore.Data, planID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Plan canceled successfully",
		})
	}
}

// HandleListPlans
func HandleListPlans() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		PlanStore.RLock()
		defer PlanStore.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(PlanStore.Data); err != nil {
			http.Error(w, "Failed to encode plan data", http.StatusInternalServerError)
		}
	}
}

// Helper function to add billing information to form data
func addBillingInfo(formData url.Values, billing *BillingInfo) {
	if billing == nil {
		return
	}

	formData.Set("first_name", billing.FirstName)
	formData.Set("last_name", billing.LastName)
	formData.Set("address1", billing.Address1)
	formData.Set("city", billing.City)
	formData.Set("state", billing.State)
	formData.Set("zip", billing.Zip)
	formData.Set("country", billing.Country)
	formData.Set("email", billing.Email)
	formData.Set("phone", billing.Phone)
}

// Helper function to generate a unique customer vault ID
func generateUniqueVaultID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil { // Just use rand.Read directly
		panic("failed to generate random number for vault ID")
	}
	return strconv.FormatInt(int64(b[0])<<56|int64(b[1])<<48|int64(b[2])<<40|int64(b[3])<<32|int64(b[4])<<24|int64(b[5])<<16|int64(b[6])<<8|int64(b[7]), 10)
}

// Helper function to send requests to NMI
func sendRequest(ctx context.Context, formData url.Values) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://secure.nmi.com/api/transact.php",
		bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return "", NewNMIError(ErrProcessingError, "failed to create request", "")
	}

	httpReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", NewNMIError(ErrNetworkError, "network error: "+err.Error(), "")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", NewNMIError(ErrProcessingError, "failed to read response", "")
	}

	return string(body), nil
}
