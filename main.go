package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/paymentintent"
)

// SuccessResponse will be returned with the client secret
type SuccessResponse struct {
	ClientSecret string `json:"clientSecret"`
}

// ErrorResponse will be used in case of errors
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PaymentRequest is the structure for the incoming request body
type PaymentRequest struct {
	Amount   int64  `json:"amount"`   // amount in smallest currency unit (e.g., cents)
	Currency string `json:"currency"` // currency in 3-letter ISO currency code (e.g., "usd")
}

// getStripeSecretKey retrieves the Stripe secret key from environment variables
func getStripeSecretKey() (string, error) {
	stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeSecretKey == "" {
		return "", fmt.Errorf("missing STRIPE_SECRET_KEY in environment variables")
	}
	return stripeSecretKey, nil
}

// createPaymentIntent interacts with Stripe's API to create a payment intent
func createPaymentIntent(amount int64, currency string) (*SuccessResponse, *ErrorResponse) {
	// First, retrieve the Stripe secret key
	stripeSecretKey, err := getStripeSecretKey()
	if err != nil {
		return nil, &ErrorResponse{
			Code:    "missing_secret_key",
			Message: err.Error(),
		}
	}

	// Set the API key for Stripe
	stripe.Key = stripeSecretKey

	// Create the payment intent
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
		PaymentMethodTypes: []*string{stripe.String("card"), stripe.String("twint")},
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			return nil, &ErrorResponse{
				Code:    string(stripeErr.Code),
				Message: stripeErr.Msg,
			}
		}
		return nil, &ErrorResponse{
			Code:    "payment_intent_creation_failed",
			Message: err.Error(),
		}
	}

	// Return the client secret in a successful response
	return &SuccessResponse{ClientSecret: pi.ClientSecret}, nil
}

// stripePaymentHandler is the handler that will be called when a payment intent is requested
func stripePaymentHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the request body to get the payment data
	var paymentRequest PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&paymentRequest); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Call createPaymentIntent to get the client secret
	successResp, errorResp := createPaymentIntent(paymentRequest.Amount, paymentRequest.Currency)

	// Handle errors
	if errorResp != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResp)
		return
	}

	// Return the success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(successResp)
}

func main() {
	// Set up the API route for handling the payment request
	http.HandleFunc("/create-payment-intent", stripePaymentHandler)

	// Use the PORT environment variable or fallback to port 8080 for local development
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server is starting on port %s...", port)

	// Start the server
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}