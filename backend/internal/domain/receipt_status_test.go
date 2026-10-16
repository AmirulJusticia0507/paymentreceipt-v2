package domain

import "testing"

func TestCanTransition_Legal(t *testing.T) {
	cases := []struct{ from, to string }{
		{"pending", "checked"},
		{"checked", "processed"},
		{"checked", "cancelled"},
		{"processed", "finished"},
		{"processed", "cancelled"},
	}
	for _, c := range cases {
		if !CanTransition(PaymentReceiptStatus(c.from), PaymentReceiptStatus(c.to)) {
			t.Errorf("CanTransition(%q,%q)=false, want true", c.from, c.to)
		}
	}
}

func TestCanTransition_Illegal(t *testing.T) {
	cases := []struct{ from, to string }{
		{"finished", "pending"},
		{"cancelled", "checked"},
		{"pending", "finished"},
		{"pending", "pending"},
		{"processed", "pending"},
		{"checked", "finished"},
	}
	for _, c := range cases {
		if CanTransition(PaymentReceiptStatus(c.from), PaymentReceiptStatus(c.to)) {
			t.Errorf("CanTransition(%q,%q)=true, want false", c.from, c.to)
		}
	}
}

func TestValidStatus(t *testing.T) {
	if !ValidStatus("pending") {
		t.Error("pending harus valid")
	}
	if ValidStatus("foo") {
		t.Error("foo harus tidak valid")
	}
}
