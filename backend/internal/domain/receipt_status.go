// Package domain berisi logika bisnis murni: aturan status payment receipt.
//
// Package ini TIDAK boleh import apa pun dari internal/* lain — murni Go standar,
// supaya bisa diuji dengan cepat tanpa database.
package domain

// PaymentReceiptStatus merepresentasikan status payment receipt sesuai enum
// payment_status di database.
type PaymentReceiptStatus string

const (
	StatusPending   PaymentReceiptStatus = "pending"
	StatusChecked   PaymentReceiptStatus = "checked"
	StatusProcessed PaymentReceiptStatus = "processed"
	StatusFinished  PaymentReceiptStatus = "finished"
	StatusCancelled PaymentReceiptStatus = "cancelled"
)

// legalTransitions adalah peta transisi status yang diizinkan.
//
//	pending → checked
//	checked → processed / cancelled
//	processed → finished / cancelled
//	finished, cancelled → (terminal)
var legalTransitions = map[PaymentReceiptStatus]map[PaymentReceiptStatus]bool{
	StatusPending:   {StatusChecked: true},
	StatusChecked:   {StatusProcessed: true, StatusCancelled: true},
	StatusProcessed: {StatusFinished: true, StatusCancelled: true},
}

// CanTransition melaporkan apakah perubahan dari -> ke diizinkan.
func CanTransition(from, to PaymentReceiptStatus) bool {
	if from == to {
		return false
	}
	targets, ok := legalTransitions[from]
	if !ok {
		return false
	}
	return targets[to]
}

// ValidStatus melaporkan apakah nilai status termasuk enum yang valid.
func ValidStatus(s string) bool {
	switch PaymentReceiptStatus(s) {
	case StatusPending, StatusChecked, StatusProcessed, StatusFinished, StatusCancelled:
		return true
	}
	return false
}
