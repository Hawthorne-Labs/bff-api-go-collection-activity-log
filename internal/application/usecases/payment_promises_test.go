package usecases

import "testing"

func TestPaymentPromisesFilterUsesPythonActivityType(t *testing.T) {
	if paymentPromiseActivityType != "promise_to_pay" {
		t.Fatalf("expected promise_to_pay, got %q", paymentPromiseActivityType)
	}
}
