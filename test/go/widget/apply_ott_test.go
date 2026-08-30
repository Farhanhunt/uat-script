package widget_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"uat-script/helper"
	widget_helper "uat-script/widget"

	"github.com/dana-id/dana-go/v2/widget/v1"
)

const widgetApplyOttCase = "ApplyOtt"

// ApplyOtt
func TestApplyOttSuccess(t *testing.T) {
	helper.RetryTest(t, 3, 1, func() error {
		caseName := "ApplyOttSuccess"

		accessToken := obtainAccessTokenViaOAuth(t, helper.TestConfig.PhoneNumber, helper.TestConfig.PIN)
		fmt.Printf("Obtained access token: %s\n", accessToken)

		jsonDict, err := helper.GetRequest(helper.TestConfig.JsonWidgetPath, widgetApplyOttCase, caseName)
		if err != nil {
			return fmt.Errorf("failed to get request data: %v", err)
		}

		if additionalInfo, ok := jsonDict["additionalInfo"].(map[string]interface{}); ok {
			additionalInfo["accessToken"] = accessToken
		} else {
			jsonDict["additionalInfo"] = map[string]interface{}{"accessToken": accessToken}
		}

		jsonBytes, err := json.Marshal(jsonDict)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %v", err)
		}

		applyOttReq := widget.ApplyOTTRequest{}
		err = json.Unmarshal(jsonBytes, &applyOttReq)
		if err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %v", err)
		}

		ctx := context.Background()
		apiResponse, httpResponse, err := helper.ApiClient.WidgetAPI.ApplyOTT(ctx).ApplyOTTRequest(applyOttReq).Execute()
		if err != nil {
			return fmt.Errorf("API request failed: %v", err)
		}
		defer httpResponse.Body.Close()

		responseJSON, err := apiResponse.MarshalJSON()
		if err != nil {
			return fmt.Errorf("failed to convert response to JSON: %v", err)
		}

		err = helper.AssertResponse(
			helper.TestConfig.JsonWidgetPath,
			widgetApplyOttCase,
			caseName,
			string(responseJSON),
			nil,
		)
		if err != nil {
			return err
		}
		return nil
	})
}

func TestApplyOttFailTokenNotFound(t *testing.T) {
	caseName := "ApplyOttCustomerTokenNotFound"
	jsonDict, err := helper.GetRequest(helper.TestConfig.JsonWidgetPath, widgetApplyOttCase, caseName)
	if err != nil {
		t.Fatalf("Failed to get request data: %v", err)
	}

	// Only patch accessToken (matching PHP: $jsonDict['additionalInfo']['accessToken'] = 'invalid_access_token_for_testing')
	if additionalInfo, ok := jsonDict["additionalInfo"].(map[string]interface{}); ok {
		additionalInfo["accessToken"] = "invalid_access_token_for_testing"
	} else {
		jsonDict["additionalInfo"] = map[string]interface{}{"accessToken": "invalid_access_token_for_testing"}
	}

	jsonBytes, err := json.Marshal(jsonDict)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	applyOttReq := widget.ApplyOTTRequest{}
	err = json.Unmarshal(jsonBytes, &applyOttReq)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	ctx := context.Background()
	_, httpResponse, err := helper.ApiClient.WidgetAPI.ApplyOTT(ctx).ApplyOTTRequest(applyOttReq).Execute()
	if err != nil {
		err = helper.AssertFailResponse(helper.TestConfig.JsonWidgetPath, widgetApplyOttCase, caseName, httpResponse, nil)
		if err != nil {
			t.Fatal(err)
		}
	} else {
		defer httpResponse.Body.Close()
		t.Fatalf("Expected ApiException was not thrown")
	}
}

func obtainAccessTokenViaOAuth(t *testing.T, phoneNumber, pin string) string {
	t.Helper()

	redirectURL, err := widget_helper.GetRedirectOauthUrl(phoneNumber, pin)
	if err != nil {
		t.Fatalf("Failed to get redirect OAuth URL: %v", err)
	}

	authCode, err := widget_helper.GetAuthCode(phoneNumber, pin, redirectURL)
	if err != nil {
		t.Fatalf("Failed to get auth code: %v", err)
	}

	ctx := context.Background()
	applyTokenReq := widget.NewApplyTokenAuthorizationCodeRequest("AUTHORIZATION_CODE", authCode)
	applyTokenReqValue := widget.ApplyTokenAuthorizationCodeRequestAsApplyTokenRequest(applyTokenReq)
	applyTokenResp, _, err := helper.ApiClient.WidgetAPI.ApplyToken(ctx).ApplyTokenRequest(applyTokenReqValue).Execute()
	if err != nil {
		t.Fatalf("Failed to obtain access token: %v", err)
	}

	return applyTokenResp.GetAccessToken()
}

func TestApplyOttCustomerAccountUserStatusAbnormal(t *testing.T) {
	helper.RetryTest(t, 3, 1, func() error {
		caseName := "ApplyOttCustomerAccountUserStatusAbnormal"

		accessToken := obtainAccessTokenViaOAuth(t, helper.TestConfig.AbnormalUserPhoneNumber, helper.TestConfig.AbnormalUserPIN)

		jsonDict, err := helper.GetRequest(helper.TestConfig.JsonWidgetPath, widgetApplyOttCase, caseName)
		if err != nil {
			return fmt.Errorf("failed to get request data: %v", err)
		}

		if additionalInfo, ok := jsonDict["additionalInfo"].(map[string]interface{}); ok {
			additionalInfo["accessToken"] = accessToken
			additionalInfo["deviceId"] = helper.TestConfig.DeviceID
		} else {
			jsonDict["additionalInfo"] = map[string]interface{}{
				"accessToken": accessToken,
				"deviceId":    helper.TestConfig.DeviceID,
			}
		}

		jsonBytes, err := json.Marshal(jsonDict)
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %v", err)
		}

		applyOttReq := widget.ApplyOTTRequest{}
		if err = json.Unmarshal(jsonBytes, &applyOttReq); err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %v", err)
		}

		ctx := context.Background()
		_, httpResponse, err := helper.ApiClient.WidgetAPI.ApplyOTT(ctx).ApplyOTTRequest(applyOttReq).Execute()
		if err != nil {
			if assertErr := helper.AssertFailResponse(helper.TestConfig.JsonWidgetPath, widgetApplyOttCase, caseName, httpResponse, nil); assertErr != nil {
				return assertErr
			}
			return nil
		}
		defer httpResponse.Body.Close()
		return fmt.Errorf("expected 4034905 error for case %s but API call succeeded", caseName)
	})
}
