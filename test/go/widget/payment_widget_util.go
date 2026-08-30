package widget

import (
	payment_gateway "uat-script/payment_gateway"
)

// PayOrder completes widget checkout via the shared PG Playwright flow (same cashier UI).
// Returns nil on success, or an error — including sandbox /n/error?errorCode=... pages.
func PayOrder(phoneNumber, pin, redirectUrl string) interface{} {
	if err := payment_gateway.PayOrder(phoneNumber, pin, redirectUrl); err != nil {
		return err
	}
	return nil
}
