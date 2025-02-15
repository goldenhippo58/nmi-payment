package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nmi-pay-int/api"
	"nmi-pay-int/config"
	"nmi-pay-int/metrics"
	"nmi-pay-int/middleware"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// LogTransaction logs transaction details to a text file
func LogTransaction(logMessage string) {
	logFile, err := os.OpenFile("logs/transactions.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		metrics.LogError(fmt.Errorf("failed to open log file: %v", err))
		return
	}
	defer logFile.Close()
	metrics.LogInfo(logMessage)
}

// SaveTransaction saves transaction details to a CSV file
func SaveTransaction(transactionID, transactionType, responseText, amount string) {
	csvFile, err := os.OpenFile("logs/transactions.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		metrics.LogError(fmt.Errorf("failed to open CSV file: %v", err))
		return
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)
	defer writer.Flush()

	// Write headers if file is empty
	fileInfo, _ := csvFile.Stat()
	if fileInfo.Size() == 0 {
		writer.Write([]string{"Timestamp", "Transaction ID", "Type", "Response", "Amount"})
	}

	writer.Write([]string{
		time.Now().Format("2006-01-02 15:04:05"),
		transactionID,
		transactionType,
		responseText,
		amount,
	})
}

func main() {
	// Initialize logger
	metrics.InitLogger()

	mode := os.Getenv("MODE")
	if mode == "serve" {
		startMicroservice()
	} else {
		runStandaloneDemo()
	}
}

func startMicroservice() {
	fmt.Println("Starting microservice...")
	cfg := config.LoadConfig()

	// Initialize router
	r := mux.NewRouter()
	fmt.Println("Router initialized...")

	// Create middleware instances
	securityMiddleware := middleware.NewSecurityMiddleware(100)

	// Apply middleware to all routes
	r.Use(middleware.LoggingMiddleware)
	r.Use(securityMiddleware.RateLimiter)
	r.Use(middleware.TimeoutMiddleware(30 * time.Second))
	r.Use(middleware.MetricsMiddleware)

	fmt.Println("Middleware applied...")

	// Add test endpoint
	r.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")

	// Payment endpoints
	r.HandleFunc("/payments/tokenize", handleTokenize(cfg)).Methods("POST")
	r.HandleFunc("/payments/sale", handleSale(cfg)).Methods("POST")
	r.HandleFunc("/payments/refund", handleRefund(cfg)).Methods("POST")
	r.HandleFunc("/payments/void", handleVoid(cfg)).Methods("POST")
	r.HandleFunc("/payments/lookup", handleLookup(cfg)).Methods("GET")

	// Recurring payment endpoints
	r.HandleFunc("/payments/recurring/create", handleCreateRecurring(cfg)).Methods("POST")
	r.HandleFunc("/payments/recurring/update/{subscription_id}", handleUpdateRecurring(cfg)).Methods("PUT")
	r.HandleFunc("/payments/recurring/cancel/{subscription_id}", handleCancelRecurring(cfg)).Methods("DELETE")

	// Plan event endpoint
	r.HandleFunc("/plans/add", api.HandleAddPlan()).Methods("POST")
	r.HandleFunc("/plans/update", api.HandleUpdatePlan()).Methods("PUT")
	r.HandleFunc("/plans/cancel/{id}", api.HandleCancelPlan()).Methods("DELETE")
	r.HandleFunc("/plans/list", api.HandleListPlans()).Methods("GET")

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// Health check endpoint
	r.HandleFunc("/health", handleHealth).Methods("GET")

	// Terminal endpoints
	r.HandleFunc("/terminal/init", handleTerminalInit(cfg)).Methods("POST")
	r.HandleFunc("/terminal/payment", handleTerminalPayment(cfg)).Methods("POST")
	r.HandleFunc("/terminal/status/{terminal_id}", handleTerminalStatus()).Methods("GET")
	r.HandleFunc("/terminal/cancel/{terminal_id}", handleTerminalCancel()).Methods("POST")

	// Print all registered routes
	fmt.Println("\nRegistered Routes:")
	r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		fmt.Printf("Route: %-30s Methods: %v\n", pathTemplate, methods)
		return nil
	})

	// Create server with timeouts
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r, // Make sure router is set as handler
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Error channel for server errors
	errChan := make(chan error, 1)

	// Start server
	fmt.Printf("\nServer starting on port %s...\n", srv.Addr)
	go func() {
		fmt.Println("Server is listening...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Server error: %v\n", err)
			errChan <- err
		}
	}()

	// Wait for either shutdown signal or server error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		fmt.Printf("Server failed: %v\n", err)
		metrics.LogError(fmt.Errorf("server failed: %v", err))
	case <-quit:
		fmt.Println("Shutdown signal received...")
		metrics.LogInfo("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			fmt.Printf("Server forced to shutdown: %v\n", err)
			metrics.LogError(fmt.Errorf("server forced to shutdown: %v", err))
		}

		fmt.Println("Server shutdown complete")
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "OK",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func handleTokenize(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.PaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		req.APIKey = cfg.APIKey
		resp, err := api.ProcessTokenization(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		LogTransaction(fmt.Sprintf("TOKENIZE: Customer Vault ID=%s, Response=SUCCESS", resp.CustomerVaultID))
	}
}

func handleSale(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		var req api.PaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		req.APIKey = cfg.APIKey
		resp, err := api.ProcessPayment(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		LogTransaction(fmt.Sprintf("SALE: Transaction ID=%s, Response=%s", resp.TransactionID, resp.ResponseText))
		SaveTransaction(resp.TransactionID, "sale", resp.ResponseText, req.Amount)
	}
}

func handleRefund(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.RefundRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		req.APIKey = cfg.APIKey
		resp, err := api.ProcessRefund(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		LogTransaction(fmt.Sprintf("REFUND: Transaction ID=%s, Response=%s", resp.TransactionID, resp.ResponseText))
		SaveTransaction(resp.TransactionID, "refund", resp.ResponseText, req.Amount)
	}
}

func handleVoid(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.VoidRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		req.APIKey = cfg.APIKey
		resp, err := api.VoidTransaction(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		LogTransaction(fmt.Sprintf("VOID: Transaction ID=%s, Response=%s", resp.TransactionID, resp.ResponseText))
		SaveTransaction(resp.TransactionID, "void", resp.ResponseText, "0.00")
	}
}

func handleLookup(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		transactionID := r.URL.Query().Get("transaction_id")
		if transactionID == "" {
			http.Error(w, "Transaction ID is required", http.StatusBadRequest)
			return
		}

		req := api.LookupRequest{
			APIKey:        cfg.APIKey,
			TransactionID: transactionID,
		}

		fmt.Printf("Lookup Request: %+v\n", req) // Debug log

		resp, err := api.LookupTransaction(r.Context(), req)
		if err != nil {
			metrics.LogError(fmt.Errorf("lookup Error: %v", err))
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("Lookup Response: %+v\n", resp) // Debug log

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		LogTransaction(fmt.Sprintf("LOOKUP: Transaction ID=%s, Response=%s", resp.TransactionID, resp.ResponseText))
	}
}

func handleCreateRecurring(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.RecurringPaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		req.APIKey = cfg.APIKey
		fmt.Printf("Received Create Recurring Request: %+v\n", req) // Debug log

		resp, err := api.ProcessRecurringPayment(r.Context(), req)
		if err != nil {
			metrics.LogError(fmt.Errorf("recurring Payment Error: %v", err))
			http.Error(w, fmt.Sprintf("Error: %v", err.Error()), http.StatusInternalServerError)
			return
		}

		fmt.Printf("Recurring Payment Response: %+v\n", resp) // Debug log

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		LogTransaction(fmt.Sprintf("RECURRING: Subscription ID=%s, Plan=%s, Response=%s", resp.SubscriptionID, req.PlanID, resp.Status))
	}
}

func handleUpdateRecurring(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		subscriptionID := vars["subscription_id"]

		if subscriptionID == "" {
			http.Error(w, "subscription_id is required", http.StatusBadRequest)
			return
		}

		var req api.RecurringPaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		req.APIKey = cfg.APIKey
		fmt.Printf("Updating Subscription ID: %s\n", subscriptionID) // Debug log

		resp, err := api.UpdateRecurringPayment(r.Context(), req, subscriptionID)
		if err != nil {
			metrics.LogError(fmt.Errorf("update Recurring Payment Error: %v", err))
			http.Error(w, fmt.Sprintf("Error: %v", err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)

		LogTransaction(fmt.Sprintf("UPDATE RECURRING: Subscription ID=%s", subscriptionID))
	}
}

func handleCancelRecurring(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		subscriptionID := vars["subscription_id"]

		err := api.CancelRecurringPayment(r.Context(), cfg.APIKey, subscriptionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Subscription cancelled successfully",
		})

		LogTransaction(fmt.Sprintf("CANCEL RECURRING: Subscription ID=%s", subscriptionID))
	}
}

func handleUpdatePlan() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var plan api.Plan
		if err := json.NewDecoder(r.Body).Decode(&plan); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		// Use api.PlanStore to manage the plan update
		api.PlanStore.Lock()
		defer api.PlanStore.Unlock()

		existingPlan, exists := api.PlanStore.Data[plan.ID]
		if !exists {
			http.Error(w, "plan not found", http.StatusNotFound)
			return
		}

		// Update fields only if they are non-empty
		if plan.Name != "" {
			existingPlan.Name = plan.Name
		}
		if plan.Amount != "" {
			existingPlan.Amount = plan.Amount
		}

		// Save the updated plan
		api.PlanStore.Data[plan.ID] = existingPlan

		// Respond with the updated plan
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(api.PlanResponse{
			Plan:    existingPlan,
			Message: "Plan updated successfully",
		})
	}
}

func handleTerminalInit(cfg *config.Config) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req api.TerminalInitRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid request payload", http.StatusBadRequest)
            return
        }

        req.APIKey = cfg.APIKey
        resp, err := api.ProcessTerminalInit(r.Context(), req)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }
}

func handleTerminalPayment(cfg *config.Config) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req api.TerminalPaymentRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "Invalid request payload", http.StatusBadRequest)
            return
        }

        req.APIKey = cfg.APIKey
        resp, err := api.ProcessTerminalPayment(r.Context(), req)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }
}

func handleTerminalStatus() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        terminalID := vars["terminal_id"]

        if terminalID == "" {
            http.Error(w, "terminal_id is required", http.StatusBadRequest)
            return
        }

        status := &api.TerminalResponse{
            Status:       "success",
            ResponseText: fmt.Sprintf("Terminal %s is active", terminalID),
            Success:      true,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(status)
    }
}

func handleTerminalCancel() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        terminalID := vars["terminal_id"]

        if terminalID == "" {
            http.Error(w, "terminal_id is required", http.StatusBadRequest)
            return
        }

        response := &api.TerminalResponse{
            Status:       "success",
            ResponseText: fmt.Sprintf("Transaction cancelled for terminal %s", terminalID),
            Success:      true,
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(response)
    }
}

// Standalone mode for testing
func runStandaloneDemo() {
	cfg := config.LoadConfig()

	// Perform a sale transaction
	paymentReq := api.PaymentRequest{
		APIKey:     cfg.APIKey,
		Amount:     "10.99",
		CreditCard: "4111111111111111",
		ExpDate:    "1225",
		CVV:        "123",
		Type:       "sale",
	}

	resp, err := api.ProcessPayment(context.Background(), paymentReq)
	if err != nil {
		metrics.LogError(fmt.Errorf("failed to process payment: %v", err))
		return
	}

	fmt.Printf("Sale Response: %+v\n", resp)
	LogTransaction(fmt.Sprintf("SALE: Transaction ID=%s, Response=%s", resp.TransactionID, resp.ResponseText))
	SaveTransaction(resp.TransactionID, "sale", resp.ResponseText, paymentReq.Amount)
}
