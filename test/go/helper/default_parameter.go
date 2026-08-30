package helper

import "os"

// TestConfig holds common test configuration variables
var TestConfig = struct {
	PhoneNumber                string
	PIN                        string
	AbnormalUserPhoneNumber    string
	AbnormalUserPIN            string
	DeviceID                   string
	MerchantID                 string
	JsonWidgetPath             string
	JsonPgPath                 string
	JsonMerchantManagementPath string
}{
	PhoneNumber:                "083811223355",
	PIN:                        "181818",
	AbnormalUserPhoneNumber:    "0855100800",
	AbnormalUserPIN:            "146838",
	DeviceID:                   "deviceid123",
	MerchantID:                 os.Getenv("MERCHANT_ID"),
	JsonWidgetPath:             "../../../resource/request/components/Widget.json",
	JsonPgPath:                 "../../../resource/request/components/PaymentGateway.json",
	JsonMerchantManagementPath: "../../../resource/request/components/MerchantManagement.json",
}
