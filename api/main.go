package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/paymentintent"
)

// SuccessResponse represents the successful response with the client secret
type SuccessResponse struct {
	ClientSecret string `json:"clientSecret"`
}

// ErrorResponse represents the error response
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PaymentRequest represents the structure for incoming payment request
type PaymentRequest struct {
	Amount   int64  `json:"amount"`   // amount in smallest currency unit (e.g., cents)
	Currency string `json:"currency"` // currency in 3-letter ISO code (e.g., "usd")
}

// getStripeSecretKey retrieves the Stripe secret key from environment variables
func getStripeSecretKey() (string, error) {
	stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeSecretKey == "" {
		return "", fmt.Errorf("missing STRIPE_SECRET_KEY in environment variables")
	}
	return stripeSecretKey, nil
}

// createPaymentIntent interacts with Stripe API to create a payment intent
func createPaymentIntent(amount int64, currency string) (*SuccessResponse, *ErrorResponse) {
	// Retrieve the Stripe secret key
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

	// Return the client secret in the successful response
	return &SuccessResponse{ClientSecret: pi.ClientSecret}, nil
}

func main() {
	r := gin.Default()

	// Root GET endpoint to confirm the server is running
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Server is up and running!",
		})
	})

	// POST /create-payment-intent to create a payment intent
	r.POST("/create-payment-intent", func(c *gin.Context) {
		var paymentRequest PaymentRequest

		// Bind JSON request to paymentRequest struct
		if err := c.ShouldBindJSON(&paymentRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Create the payment intent
		successResp, errorResp := createPaymentIntent(paymentRequest.Amount, paymentRequest.Currency)

		// Handle errors
		if errorResp != nil {
			c.JSON(http.StatusInternalServerError, errorResp)
			return
		}

		// Return the success response
		c.JSON(http.StatusOK, successResp)
	})

	// Use the PORT environment variable or fallback to port 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server is starting on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
