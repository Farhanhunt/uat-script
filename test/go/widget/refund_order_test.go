package widget_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	widget "github.com/dana-id/dana-go/v2/widget/v1"
	"github.com/google/uuid"

	"uat-script/helper"
)

const (
	refundOrderTitleCase      = "RefundOrder"
	refundOrderJsonPath       = "../../../resource/request/components/Widget.json"
	paymentForRefundTitleCase = "Payment"
)

// createTestWidgetPaymentForRefund creates a test widget payment that can be refunded
func createTestWidgetPaymentForRefund() (string, error) {
	var partnerReferenceNo string
	result, err := helper.RetryOnInconsistentRequest(func() (interface{}, error) {
		// Get the request data from the Payment JSON section
		caseName := "PaymentSuccess"
		jsonDict, err := helper.GetRequest(refundOrderJsonPath, paymentForRefundTitleCase, caseName)
		if err != nil {
			return "", err
		}

		// Set a unique partner reference number
		partnerReferenceNo = uuid.New().String()
		jsonDict["partnerReferenceNo"] = partnerReferenceNo
		jsonDict["validUpTo"] = helper.GenerateFormattedDate(900, 7)

		// Create the WidgetPaymentRequest object and populate it with JSON data
		jsonBytes, err := json.Marshal(jsonDict)
		if err != nil {
			return "", err
		}

		var request widget.WidgetPaymentRequest
		err = json.Unmarshal(jsonBytes, &request)
		if err != nil {
			return "", err
		}

		// Make the API call to create a payment
		ctx := context.Background()
		_, httpResponse, err := helper.ApiClient.WidgetAPI.WidgetPayment(ctx).WidgetPaymentRequest(request).Execute()
		if err != nil {
			return "", err
		}
		defer httpResponse.Body.Close()

		return partnerReferenceNo, nil
	}, 3, 2*time.Second)
	if err != nil {
		return "", err
	}
	return result.(string), nil
}

// RefundOrder
func TestRefundOrderValidScenario(t *testing.T) {
	helper.RetryTest(t, 3, 1, func() error {
		partnerReferenceNo, err := createTestWidgetPaymentPaid()
		if err != nil {
			return fmt.Errorf("failed to create paid widget payment: %w", err)
		}

		time.Sleep(2 * time.Second)

		caseName := "RefundOrderValidScenario"
		jsonDict, err := helper.GetRequest(refundOrderJsonPath, refundOrderTitleCase, caseName)
		if err != nil {
			return fmt.Errorf("failed to get request data: %w", err)
		}

		jsonDict["originalPartnerReferenceNo"] = partnerReferenceNo
		jsonDict["partnerRefundNo"] = partnerReferenceNo
		jsonDict["merchantId"] = helper.TestConfig.MerchantID

		jsonBytes, err := json.Marshal(jsonDict)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		var request widget.RefundOrderRequest
		if err = json.Unmarshal(jsonBytes, &request); err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		ctx := context.Background()
		apiResponse, httpResponse, err := helper.ApiClient.WidgetAPI.RefundOrder(ctx).RefundOrderRequest(request).Execute()
		if err != nil {
			return fmt.Errorf("API request failed: %w", err)
		}
		defer httpResponse.Body.Close()

		responseJSON, err := apiResponse.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to convert response to JSON: %w", err)
		}

		variableDict := map[string]interface{}{
			"partnerReferenceNo": partnerReferenceNo,
		}

		if err = helper.AssertResponse(
			refundOrderJsonPath,
			refundOrderTitleCase,
			caseName,
			string(responseJSON),
			variableDict,
		); err != nil {
			return err
		}
		return nil
	})
}
func TestRefundInProcess(t *testing.T) {
	t.Skip("Skip: Requires a paid order to refund, which needs complex setup with payment completion")
	caseName := "RefundInProcess"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(refundOrderJsonPath, refundOrderTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.RefundOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute the SDK API call
	ctx := context.Background()
	apiResponse, httpResponse, err := helper.ApiClient.WidgetAPI.RefundOrder(ctx).RefundOrderRequest(request).Execute()
	if err != nil {
		t.Fatalf("API request failed: %v", err)
	}
	defer httpResponse.Body.Close()

	// Convert the response to JSON for assertion
	responseJSON, err := apiResponse.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to convert response to JSON: %v", err)
	}

	// Assert the success response
	variableDict := map[string]interface{}{
		"partnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
	}

	err = helper.AssertResponse(
		refundOrderJsonPath,
		refundOrderTitleCase,
		caseName,
		string(responseJSON),
		variableDict,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRefundFailDuplicateRequest(t *testing.T) {
	helper.RetryTest(t, 3, 1, func() error {
		partnerReferenceNo, err := createTestWidgetPaymentPaid()
		if err != nil {
			return fmt.Errorf("failed to create paid widget payment: %w", err)
		}

		time.Sleep(2 * time.Second)

		caseName := "RefundFailDuplicateRequest"
		jsonDict, err := helper.GetRequest(refundOrderJsonPath, refundOrderTitleCase, caseName)
		if err != nil {
			return fmt.Errorf("failed to get request data: %w", err)
		}

		jsonDict["originalPartnerReferenceNo"] = partnerReferenceNo
		jsonDict["partnerRefundNo"] = partnerReferenceNo
		jsonDict["merchantId"] = helper.TestConfig.MerchantID

		jsonBytes, err := json.Marshal(jsonDict)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		var request widget.RefundOrderRequest
		if err = json.Unmarshal(jsonBytes, &request); err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		ctx := context.Background()

		// First refund with this partnerRefundNo — should succeed.
		_, firstHTTP, err := helper.ApiClient.WidgetAPI.RefundOrder(ctx).RefundOrderRequest(request).Execute()
		if err != nil {
			return fmt.Errorf("first refund should succeed before duplicate attempt: %w", err)
		}
		firstHTTP.Body.Close()

		time.Sleep(2 * time.Second)

		// Same partnerRefundNo, different refundAmount → Inconsistent Request (4045818).
		request.SetRefundAmount(widget.Money{Value: "2.00", Currency: "IDR"})

		_, httpResponse, err := helper.ApiClient.WidgetAPI.RefundOrder(ctx).RefundOrderRequest(request).Execute()
		if err != nil {
			variableDict := map[string]interface{}{
				"partnerReferenceNo": partnerReferenceNo,
			}
			if err = helper.AssertFailResponse(refundOrderJsonPath, refundOrderTitleCase, caseName, httpResponse, variableDict); err != nil {
				return err
			}
			return nil
		}

		httpResponse.Body.Close()
		return fmt.Errorf("expected duplicate refund error for case %s but API call succeeded", caseName)
	})
}
func TestRefundFailOrderNotPaid(t *testing.T) {
	caseName := "RefundFailOrderNotPaid"

	// Create a test payment first (which will be in INIT status, not paid)
	partnerReferenceNo, err := createTestWidgetPaymentForRefund()
	if err != nil {
		t.Fatalf("Failed to create test widget payment: %v", err)
	}

	// Give time for the payment to be processed
	time.Sleep(2 * time.Second)

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(refundOrderJsonPath, refundOrderTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Set the partner reference number from the created payment
	jsonDict["originalPartnerReferenceNo"] = partnerReferenceNo
	jsonDict["partnerRefundNo"] = partnerReferenceNo

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.RefundOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute the SDK API call and expect error response
	ctx := context.Background()
	_, httpResponse, err := helper.ApiClient.WidgetAPI.RefundOrder(ctx).RefundOrderRequest(request).Execute()
	if err != nil {
		// This is expected for error test cases
		variableDict := map[string]interface{}{
			"partnerReferenceNo": partnerReferenceNo,
		}

		// Assert the error response matches expected error pattern
		err = helper.AssertFailResponse(refundOrderJsonPath, refundOrderTitleCase, caseName, httpResponse, variableDict)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		// If no error occurred, this is unexpected for error test cases
		defer httpResponse.Body.Close()
		t.Fatalf("Expected error for case %s but API call succeeded", caseName)
	}
}

func TestRefundFailMandatoryParameterInvalid(t *testing.T) {
	caseName := "RefundFailMandatoryParameterInvalid"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(refundOrderJsonPath, refundOrderTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.RefundOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute API request with missing mandatory field to trigger validation error
	ctx := context.Background()
	endpoint := "https://api.sandbox.dana.id/v1.0/debit/refund.htm"
	resourcePath := "/v1.0/debit/refund.htm"

	// Set custom headers with empty timestamp to trigger invalid mandatory field error
	customHeaders := map[string]string{
		"X-TIMESTAMP": "",
	}

	variableDict := map[string]interface{}{
		"partnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
	}

	err = helper.ExecuteAndAssertErrorResponse(
		t,
		ctx,
		&request,
		"POST",
		endpoint,
		resourcePath,
		refundOrderJsonPath,
		refundOrderTitleCase,
		caseName,
		customHeaders,
		variableDict,
	)
	if err != nil {
		t.Fatal(err)
	}
}
func TestRefundFailOrderNotExist(t *testing.T) {
	caseName := "RefundFailOrderNotExist"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(refundOrderJsonPath, refundOrderTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Use the original non-existent partner reference number from the JSON
	// This should trigger the "Invalid Bill" error

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.RefundOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute the SDK API call and expect error response
	ctx := context.Background()
	_, httpResponse, err := helper.ApiClient.WidgetAPI.RefundOrder(ctx).RefundOrderRequest(request).Execute()
	if err != nil {
		// This is expected for error test cases
		variableDict := map[string]interface{}{
			"partnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
		}

		// Assert the error response matches expected error pattern
		err = helper.AssertFailResponse(refundOrderJsonPath, refundOrderTitleCase, caseName, httpResponse, variableDict)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		// If no error occurred, this is unexpected for error test cases
		defer httpResponse.Body.Close()
		t.Fatalf("Expected error for case %s but API call succeeded", caseName)
	}
}
func TestRefundFailInvalidSignature(t *testing.T) {
	caseName := "RefundFailInvalidSignature"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(refundOrderJsonPath, refundOrderTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.RefundOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute API request with invalid signature to trigger unauthorized error
	ctx := context.Background()
	endpoint := "https://api.sandbox.dana.id/v1.0/debit/refund.htm"
	resourcePath := "/v1.0/debit/refund.htm"

	// Set custom headers with invalid signature to trigger unauthorized error
	customHeaders := map[string]string{
		"X-SIGNATURE": "invalid_signature",
	}

	variableDict := map[string]interface{}{
		"partnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
	}

	err = helper.ExecuteAndAssertErrorResponse(
		t,
		ctx,
		&request,
		"POST",
		endpoint,
		resourcePath,
		refundOrderJsonPath,
		refundOrderTitleCase,
		caseName,
		customHeaders,
		variableDict,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestRefundFailTimeout(t *testing.T) {
	caseName := "RefundFailTimeout"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(refundOrderJsonPath, refundOrderTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.RefundOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute the SDK API call and expect error response
	ctx := context.Background()
	_, httpResponse, err := helper.ApiClient.WidgetAPI.RefundOrder(ctx).RefundOrderRequest(request).Execute()
	if err != nil {
		// This is expected for error test cases
		variableDict := map[string]interface{}{
			"partnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
		}

		// Assert the error response matches expected error pattern
		err = helper.AssertFailResponse(refundOrderJsonPath, refundOrderTitleCase, caseName, httpResponse, variableDict)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		// If no error occurred, this is unexpected for error test cases
		defer httpResponse.Body.Close()
		t.Fatalf("Expected error for case %s but API call succeeded", caseName)
	}
}
func TestRefundFailIdempotent(t *testing.T) {
	t.Skip("Skip: Idempotent test may require special handling or actual payment scenario")
	caseName := "RefundFailIdempotent"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(refundOrderJsonPath, refundOrderTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.RefundOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute the SDK API call and expect success response (idempotent case)
	ctx := context.Background()
	apiResponse, httpResponse, err := helper.ApiClient.WidgetAPI.RefundOrder(ctx).RefundOrderRequest(request).Execute()
	if err != nil {
		t.Fatalf("API request failed: %v", err)
	}
	defer httpResponse.Body.Close()

	// Convert the response to JSON for assertion
	responseJSON, err := apiResponse.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to convert response to JSON: %v", err)
	}

	// Assert the success response
	variableDict := map[string]interface{}{
		"partnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
	}

	err = helper.AssertResponse(
		refundOrderJsonPath,
		refundOrderTitleCase,
		caseName,
		string(responseJSON),
		variableDict,
	)
	if err != nil {
		t.Fatal(err)
	}
}
func TestRefundFailMerchantStatusAbnormal(t *testing.T) {
	caseName := "RefundFailMerchantStatusAbnormal"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(refundOrderJsonPath, refundOrderTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.RefundOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute the SDK API call and expect error response
	ctx := context.Background()
	_, httpResponse, err := helper.ApiClient.WidgetAPI.RefundOrder(ctx).RefundOrderRequest(request).Execute()
	if err != nil {
		// This is expected for error test cases
		variableDict := map[string]interface{}{
			"partnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
		}

		// Assert the error response matches expected error pattern
		err = helper.AssertFailResponse(refundOrderJsonPath, refundOrderTitleCase, caseName, httpResponse, variableDict)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		// If no error occurred, this is unexpected for error test cases
		defer httpResponse.Body.Close()
		t.Fatalf("Expected error for case %s but API call succeeded", caseName)
	}
}
