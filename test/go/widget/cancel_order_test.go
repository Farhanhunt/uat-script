package widget_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	widget "github.com/dana-id/dana-go/v2/widget/v1"
	"github.com/google/uuid"

	"uat-script/helper"
	widget_helper "uat-script/widget"
)

const (
	widgetTitleCase = "CancelOrder"
	widgetJsonPath  = "../../../resource/request/components/Widget.json"
)

func TestCancelOrderValidScenario(t *testing.T) {
	partnerReferenceNo, err := createTestWidgetPayment()
	if err != nil {
		t.Fatalf("Failed to create test widget payment: %v", err)
	}

	time.Sleep(2 * time.Second)

	caseName := "CancelOrderValidScenario"
	jsonDict, err := helper.GetRequest(widgetJsonPath, widgetTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	jsonDict["originalPartnerReferenceNo"] = partnerReferenceNo
	jsonDict["merchantId"] = os.Getenv("MERCHANT_ID")

	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.CancelOrderRequest
	if err = json.Unmarshal(jsonBytes, &request); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	ctx := context.Background()
	apiResponse, httpResponse, err := helper.ApiClient.WidgetAPI.CancelOrder(ctx).CancelOrderRequest(request).Execute()
	if err != nil {
		t.Fatalf("API call failed: %v", err)
	}
	defer httpResponse.Body.Close()

	responseJSON, err := apiResponse.MarshalJSON()
	if err != nil {
		t.Fatalf("Failed to convert response to JSON: %v", err)
	}

	if err = helper.AssertResponse(
		widgetJsonPath,
		widgetTitleCase,
		caseName,
		string(responseJSON),
		map[string]interface{}{"partnerReferenceNo": partnerReferenceNo},
	); err != nil {
		t.Fatal(err)
	}
}

func TestCancelOrderFailUserStatusAbnormal(t *testing.T) {
	caseName := "CancelOrderFailUserStatusAbnormal"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(widgetJsonPath, widgetTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.CancelOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute the SDK API call and expect error response
	ctx := context.Background()

	// Make the API call using the Widget SDK
	_, httpResponse, err := helper.ApiClient.WidgetAPI.CancelOrder(ctx).CancelOrderRequest(request).Execute()
	if err != nil {
		// This is expected for error test cases
		variableDict := map[string]interface{}{
			"originalPartnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
		}

		// Assert the error response matches expected error pattern
		err = helper.AssertFailResponse(widgetJsonPath, widgetTitleCase, caseName, httpResponse, variableDict)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		// If no error occurred, this is unexpected for error test cases
		defer httpResponse.Body.Close()
		t.Fatalf("Expected error for case %s but API call succeeded", caseName)
	}
}

func TestCancelOrderFailMerchantStatusAbnormal(t *testing.T) {
	caseName := "CancelOrderFailMerchantStatusAbnormal"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(widgetJsonPath, widgetTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.CancelOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute the SDK API call and expect error response
	ctx := context.Background()

	// Make the API call using the Widget SDK
	_, httpResponse, err := helper.ApiClient.WidgetAPI.CancelOrder(ctx).CancelOrderRequest(request).Execute()
	if err != nil {
		// This is expected for error test cases
		variableDict := map[string]interface{}{
			"originalPartnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
		}

		// Assert the error response matches expected error pattern
		err = helper.AssertFailResponse(widgetJsonPath, widgetTitleCase, caseName, httpResponse, variableDict)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		// If no error occurred, this is unexpected for error test cases
		defer httpResponse.Body.Close()
		t.Fatalf("Expected error for case %s but API call succeeded", caseName)
	}
}

func TestCancelOrderFailMissingParameter(t *testing.T) {
	caseName := "CancelOrderFailMissingParameter"

	jsonDict, err := helper.GetRequest(widgetJsonPath, widgetTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	jsonDict["merchantId"] = helper.TestConfig.MerchantID

	ctx := context.Background()
	endpoint := "https://api.sandbox.dana.id/v1.0/debit/cancel.htm"
	resourcePath := "/v1.0/debit/cancel.htm"

	if err = helper.ExecuteAndAssertErrorResponse(
		t,
		ctx,
		jsonDict,
		"POST",
		endpoint,
		resourcePath,
		widgetJsonPath,
		widgetTitleCase,
		caseName,
		nil,
		nil,
	); err != nil {
		t.Fatal(err)
	}
}

func TestCancelOrderFailOrderNotExist(t *testing.T) {
	caseName := "CancelOrderFailOrderNotExist"

	helper.RetryTest(t, 3, time.Second, func() error {
		jsonDict, err := helper.GetRequest(widgetJsonPath, widgetTitleCase, caseName)
		if err != nil {
			return fmt.Errorf("failed to get request data: %w", err)
		}

		partnerReferenceNo := uuid.New().String()
		referenceNo := uuid.New().String()
		jsonDict["originalPartnerReferenceNo"] = partnerReferenceNo
		jsonDict["originalReferenceNo"] = referenceNo
		jsonDict["merchantId"] = os.Getenv("MERCHANT_ID")

		jsonBytes, err := json.Marshal(jsonDict)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		var request widget.CancelOrderRequest
		if err = json.Unmarshal(jsonBytes, &request); err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		ctx := context.Background()
		apiResponse, httpResponse, err := helper.ApiClient.WidgetAPI.CancelOrder(ctx).CancelOrderRequest(request).Execute()

		variableDict := map[string]interface{}{
			"originalPartnerReferenceNo": partnerReferenceNo,
		}
		return helper.AssertSdkErrorResponse(widgetJsonPath, widgetTitleCase, caseName, apiResponse, httpResponse, err, variableDict)
	})
}

func TestCancelOrderFailExceedCancelWindowTime(t *testing.T) {
	caseName := "CancelOrderFailExceedCancelWindowTime"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(widgetJsonPath, widgetTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.CancelOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Execute the SDK API call and expect error response
	ctx := context.Background()

	// Make the API call using the Widget SDK
	_, httpResponse, err := helper.ApiClient.WidgetAPI.CancelOrder(ctx).CancelOrderRequest(request).Execute()
	if err != nil {
		// This is expected for error test cases
		variableDict := map[string]interface{}{
			"originalPartnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
		}

		// Assert the error response matches expected error pattern
		err = helper.AssertFailResponse(widgetJsonPath, widgetTitleCase, caseName, httpResponse, variableDict)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		// If no error occurred, this is unexpected for error test cases
		defer httpResponse.Body.Close()
		t.Fatalf("Expected error for case %s but API call succeeded", caseName)
	}
}

func TestCancelOrderFailTimeout(t *testing.T) {
	caseName := "CancelOrderFailTimeout"

	// Get the request data from JSON
	jsonDict, err := helper.GetRequest(widgetJsonPath, widgetTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Marshal to JSON and unmarshal to widget SDK struct
	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.CancelOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	ctx := context.Background()

	_, httpResponse, err := helper.ApiClient.WidgetAPI.CancelOrder(ctx).CancelOrderRequest(request).Execute()
	if err != nil {
		variableDict := map[string]interface{}{
			"originalPartnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
		}
		err = helper.AssertFailResponse(widgetJsonPath, widgetTitleCase, caseName, httpResponse, variableDict)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		defer httpResponse.Body.Close()
		t.Fatalf("Expected error for case %s but API call succeeded", caseName)
	}
}

func TestCancelOrderFailAccountStatusAbnormal(t *testing.T) {
	caseName := "CancelOrderFailAccountStatusAbnormal"

	jsonDict, err := helper.GetRequest(widgetJsonPath, widgetTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.CancelOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	ctx := context.Background()
	_, httpResponse, err := helper.ApiClient.WidgetAPI.CancelOrder(ctx).CancelOrderRequest(request).Execute()
	if err != nil {
		variableDict := map[string]interface{}{
			"originalPartnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
		}
		err = helper.AssertFailResponse(widgetJsonPath, widgetTitleCase, caseName, httpResponse, variableDict)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		defer httpResponse.Body.Close()
		t.Fatalf("Expected error for case %s but API call succeeded", caseName)
	}
}

func TestCancelOrderFailInsufficientMerchantBalance(t *testing.T) {
	caseName := "CancelOrderFailInsufficientMerchantBalance"

	jsonDict, err := helper.GetRequest(widgetJsonPath, widgetTitleCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var request widget.CancelOrderRequest
	err = json.Unmarshal(jsonBytes, &request)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	ctx := context.Background()
	_, httpResponse, err := helper.ApiClient.WidgetAPI.CancelOrder(ctx).CancelOrderRequest(request).Execute()
	if err != nil {
		variableDict := map[string]interface{}{
			"originalPartnerReferenceNo": jsonDict["originalPartnerReferenceNo"],
		}
		err = helper.AssertFailResponse(widgetJsonPath, widgetTitleCase, caseName, httpResponse, variableDict)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		defer httpResponse.Body.Close()
		t.Fatalf("Expected error for case %s but API call succeeded", caseName)
	}
}

func TestCancelOrderFailOrderInvalidStatus(t *testing.T) {
	helper.RetryTest(t, 3, 1, func() error {
		partnerReferenceNo, err := createTestWidgetPaymentRefunded()
		if err != nil {
			return fmt.Errorf("failed to create paid and refunded widget payment: %w", err)
		}

		time.Sleep(2 * time.Second)

		caseName := "CancelOrderFailOrderInvalidStatus"
		jsonDict, err := helper.GetRequest(widgetJsonPath, widgetTitleCase, caseName)
		if err != nil {
			return fmt.Errorf("failed to get request data: %w", err)
		}

		jsonDict["originalPartnerReferenceNo"] = partnerReferenceNo
		jsonDict["merchantId"] = helper.TestConfig.MerchantID

		jsonBytes, err := json.Marshal(jsonDict)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}

		var request widget.CancelOrderRequest
		if err = json.Unmarshal(jsonBytes, &request); err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}

		ctx := context.Background()
		_, httpResponse, err := helper.ApiClient.WidgetAPI.CancelOrder(ctx).CancelOrderRequest(request).Execute()
		if err != nil {
			variableDict := map[string]interface{}{
				"originalPartnerReferenceNo": partnerReferenceNo,
			}
			if err = helper.AssertFailResponse(widgetJsonPath, widgetTitleCase, caseName, httpResponse, variableDict); err != nil {
				return err
			}
			return nil
		}

		httpResponse.Body.Close()
		return fmt.Errorf("expected error for case %s but API call succeeded", caseName)
	})
}

func createTestWidgetPaymentWithRedirect() (string, string, error) {
	var partnerReferenceNo, webRedirectUrl string
	_, err := helper.RetryOnInconsistentRequest(func() (interface{}, error) {
		jsonDict, err := helper.GetRequest(widgetJsonPath, "Payment", "PaymentSuccess")
		if err != nil {
			return nil, err
		}

		partnerReferenceNo = uuid.New().String()
		jsonDict["partnerReferenceNo"] = partnerReferenceNo
		jsonDict["validUpTo"] = helper.GenerateFormattedDate(900, 7)
		jsonDict["merchantId"] = helper.TestConfig.MerchantID

		jsonBytes, err := json.Marshal(jsonDict)
		if err != nil {
			return nil, err
		}

		var request widget.WidgetPaymentRequest
		if err = json.Unmarshal(jsonBytes, &request); err != nil {
			return nil, err
		}

		ctx := context.Background()
		_, httpResponse, err := helper.ApiClient.WidgetAPI.WidgetPayment(ctx).WidgetPaymentRequest(request).Execute()
		if err != nil {
			return nil, err
		}

		webRedirectUrl, err = helper.GetValueFromResponseBody(httpResponse, "webRedirectUrl")
		httpResponse.Body.Close()
		if err != nil {
			return nil, err
		}

		return partnerReferenceNo, nil
	}, 3, 2*time.Second)
	if err != nil {
		return "", "", err
	}
	return partnerReferenceNo, webRedirectUrl, nil
}

func createTestWidgetPaymentPaid() (string, error) {
	partnerReferenceNo, webRedirectUrl, err := createTestWidgetPaymentWithRedirect()
	if err != nil {
		return "", err
	}

	if payResult := widget_helper.PayOrder(helper.TestConfig.PhoneNumber, helper.TestConfig.PIN, webRedirectUrl); payResult != nil {
		if payErr, ok := payResult.(error); ok {
			return "", payErr
		}
		return "", fmt.Errorf("widget payment automation failed: %v", payResult)
	}

	time.Sleep(5 * time.Second)
	return partnerReferenceNo, nil
}

func createTestWidgetPaymentRefunded() (string, error) {
	result, err := helper.RetryOnInconsistentRequest(func() (interface{}, error) {
		partnerReferenceNo, err := createTestWidgetPaymentPaid()
		if err != nil {
			return "", err
		}

		jsonDict, err := helper.GetRequest(widgetJsonPath, "RefundOrder", "RefundOrderValidScenario")
		if err != nil {
			return "", err
		}

		jsonDict["originalPartnerReferenceNo"] = partnerReferenceNo
		jsonDict["partnerRefundNo"] = partnerReferenceNo
		jsonDict["merchantId"] = helper.TestConfig.MerchantID

		jsonBytes, err := json.Marshal(jsonDict)
		if err != nil {
			return "", err
		}

		var request widget.RefundOrderRequest
		if err = json.Unmarshal(jsonBytes, &request); err != nil {
			return "", err
		}

		ctx := context.Background()
		_, httpResponse, err := helper.ApiClient.WidgetAPI.RefundOrder(ctx).RefundOrderRequest(request).Execute()
		if err != nil {
			return "", err
		}
		httpResponse.Body.Close()

		return partnerReferenceNo, nil
	}, 3, 2*time.Second)
	if err != nil {
		return "", err
	}
	return result.(string), nil
}
