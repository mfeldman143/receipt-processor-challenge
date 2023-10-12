package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	ErrInvalidReceiptFormat = "Invalid receipt format"
	ErrInvalidReceiptData   = "Invalid receipt data"
	ErrReceiptNotFound      = "Receipt not found"
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

type ReceiptPoints struct {
	ID     string `json:"id"`
	Points int    `json:"points"`
}

var receiptStorage = make(map[string]int)
var mu sync.Mutex

func calculatePoints(receipt Receipt) int {
	points := 0

	alphaNumericCount := len(regexp.MustCompile(`[a-zA-Z0-9]`).FindAllString(receipt.Retailer, -1))
	points += alphaNumericCount

	totalValue, err := strconv.ParseFloat(receipt.Total, 64)
	if err == nil {
		if totalValue == float64(int(totalValue)) {
			points += 50
		}

		if int(totalValue*100)%25 == 0 {
			points += 25
		}
	}
	points += (len(receipt.Items) / 2) * 5

	for _, item := range receipt.Items {
		itemPrice, err := strconv.ParseFloat(item.Price, 64)
		if err != nil {
			continue // If there's an error, skip processing this item
		}
		if len(strings.TrimSpace(item.ShortDescription))%3 == 0 {
			points += int(math.Ceil(itemPrice * 0.2)) // Use math.Ceil to round up
		}
	}

	parsedDate, _ := time.Parse("2006-01-02", receipt.PurchaseDate)
	if parsedDate.Day()%2 == 1 {
		points += 6
	}

	parsedTime, _ := time.Parse("15:04", receipt.PurchaseTime)
	if parsedTime.Hour() >= 14 && parsedTime.Hour() < 16 {
		points += 10
	}

	return points
}

func isValidReceipt(receipt Receipt) bool {
	if receipt.Retailer == "" || !regexp.MustCompile(`^[\w\s\-&]+$`).MatchString(receipt.Retailer) {
		return false
	}
	if len(receipt.Items) == 0 {
		return false
	}
	for _, item := range receipt.Items {
		if !isValidItem(item) {
			return false
		}
	}
	return true
}

func isValidItem(item Item) bool {
	if item.ShortDescription == "" || !regexp.MustCompile(`^[\w\s\-]+$`).MatchString(item.ShortDescription) {
		return false
	}
	itemPrice, err := strconv.ParseFloat(item.Price, 64)
	if err != nil || itemPrice <= 0 {
		return false
	}
	return true
}

func processReceiptHandler(w http.ResponseWriter, r *http.Request) {
	var receipt Receipt
	err := json.NewDecoder(r.Body).Decode(&receipt)
	if err != nil {
		http.Error(w, ErrInvalidReceiptFormat, http.StatusBadRequest)
		return
	}
	if !isValidReceipt(receipt) {
		http.Error(w, ErrInvalidReceiptData, http.StatusBadRequest)
		return
	}

	points := calculatePoints(receipt)
	id := uuid.New().String() //id := time.Now().Format(time.RFC3339Nano)

	mu.Lock()
	receiptStorage[id] = points
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func getPointsHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/receipts/"):strings.LastIndex(r.URL.Path, "/points")]

	mu.Lock()
	points, exists := receiptStorage[id]
	mu.Unlock()

	if !exists {
		http.Error(w, ErrReceiptNotFound, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"points": points})
}

func main() {
	http.HandleFunc("/receipts/process", processReceiptHandler)
	http.HandleFunc("/receipts/", getPointsHandler)

	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}
