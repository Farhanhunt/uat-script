<?php

namespace DanaUat\Widget;

use PHPUnit\Framework\TestCase;
use Dana\Widget\v1\Api\WidgetApi;
use Dana\Configuration;
use Dana\ObjectSerializer;
use Dana\Env;
use Dana\ApiException;
use DanaUat\Helper\Assertion;
use DanaUat\Helper\Util;
use Exception;

class ApplyTokenTest extends TestCase
{
    private static $titleCase = 'ApplyToken';
    private static $jsonPathFile = 'resource/request/components/Widget.json';
    private static $apiInstance;
    private static $phoneNumber = '083811223355';
    private static $userPin = '181818';
    private static $deviceId = 'deviceid123';
    private static $authCode;
    private static $partnerId;

    public static function setUpBeforeClass(): void
    {
        $configuration = new Configuration();
        $configuration->setApiKey('PRIVATE_KEY', getenv('PRIVATE_KEY'));
        $configuration->setApiKey('ORIGIN', getenv('ORIGIN'));
        $configuration->setApiKey('X_PARTNER_ID', getenv('X_PARTNER_ID'));
        $configuration->setApiKey('ENV', Env::SANDBOX);
        self::$apiInstance = new WidgetApi(null, $configuration);

        self::$authCode = OauthUtil::getAuthCode(
            getenv('X_PARTNER_ID'),
            null,
            self::$phoneNumber,
            self::$userPin,
            getenv('REDIRECT_URL_OAUTH')
        );
        self::$partnerId = getenv('X_PARTNER_ID');
    }

    /**
     * Should give success response for apply token scenario
     */
    public function testApplyTokenSuccess(): void
    {   
        Util::withDelay(function () {
            // Continue with your API call
            $caseName = 'ApplyTokenSuccess';
            $jsonDict = Util::getRequest(
                self::$jsonPathFile,
                self::$titleCase,
                $caseName
            );

            $requestObj = ObjectSerializer::deserialize(
                $jsonDict,
                'Dana\Widget\v1\Model\ApplyTokenRequest'
            );

            $requestObj->setAuthCode(self::$authCode);

            try {
                $apiResponse = self::$apiInstance->applyToken($requestObj);
                Assertion::assertResponse(
                    self::$jsonPathFile,
                    self::$titleCase,
                    $caseName,
                    $apiResponse->__toString()
                );
            } catch (ApiException $e) {
                $this->fail('Failed to cancel order in progress: ' . $e->getMessage());
            } catch (Exception $e) {
                $this->fail('Unexpected exception: ' . $e->getMessage());
            }
        });
    }

    /**
     * Test apply token failure when auth code has already been used
     * This test verifies that using an auth code twice fails with appropriate error
     */
    public function testApplyTokenFailAuthcodeUsed(): void
    {
        Util::withDelay(function () {
            $caseName = 'ApplyTokenFailAuthcodeUsed';
            
            // Get a fresh auth code for this test
            $freshAuthCode = OauthUtil::getAuthCode(
                self::$partnerId,
                null,
                self::$phoneNumber,
                self::$userPin,
                getenv('REDIRECT_URL_OAUTH')
            );
            
            $jsonDict = Util::getRequest(
                self::$jsonPathFile,
                self::$titleCase,
                $caseName
            );

            $requestObj = ObjectSerializer::deserialize(
                $jsonDict,
                'Dana\Widget\v1\Model\ApplyTokenRequest'
            );

            $requestObj->setAuthCode($freshAuthCode);

            try {
                // First call - should succeed and consume the auth code
                $firstResponse = self::$apiInstance->applyToken($requestObj);
                echo "First apply token call succeeded, auth code consumed\n";
                
                // Second call with the same auth code - should fail
                $secondResponse = self::$apiInstance->applyToken($requestObj);
                $this->fail('Expected ApiException for used auth code but the API call succeeded');
                
            } catch (ApiException $e) {
                // We expect this to fail on the second call
                echo "Apply token failed as expected with used auth code: " . $e->getMessage() . "\n";
                
                // Get the response body from the exception
                $responseContent = (string)$e->getResponseBody();

                // Use assertFailResponse to validate the error response
                Assertion::assertFailResponse(
                    self::$jsonPathFile,
                    self::$titleCase,
                    $caseName,
                    $responseContent
                );
                $this->assertTrue(true);
            } catch (Exception $e) {
                $this->fail('Unexpected exception: ' . $e->getMessage());
            }
        });
    }

    public function testApplyTokenFailInvalidSignature(): void
    {   
        Util::withDelay(function () {
            // Continue with your API call
            $caseName = 'ApplyTokenFailInvalidSignature';
            $jsonDict = Util::getRequest(
                self::$jsonPathFile,
                self::$titleCase,
                $caseName
            );

            $headers = Util::getHeadersWithSignature(
                'POST',
                "/v1.0/access-token/b2b2c.htm",
                $jsonDict,
                true,
                false,
                true
            );

            $requestObj = ObjectSerializer::deserialize(
                $jsonDict,
                'Dana\Widget\v1\Model\ApplyTokenRequest'
            );

            $headers['X-CLIENT-KEY'] = self::$partnerId;

            try {
                Util::executeApiRequest(
                    'POST',
                    'https://api.sandbox.dana.id/v1.0/access-token/b2b2c.htm',
                    $headers,
                    $requestObj
                );

                $this->fail('Expected ApiException for invalid signature but the API call succeeded');
            } catch (ApiException $e) {
                // We expect a 401 Unauthorized for invalid signature
                $this->assertEquals(401, $e->getCode(), "Expected HTTP 401 Unauthorized for invalid signature, got {$e->getCode()}");

                // Get the response body from the exception
                $responseContent = (string)$e->getResponseBody();

                // Use assertFailResponse to validate the error response
                Assertion::assertFailResponse(
                    self::$jsonPathFile,
                    self::$titleCase,
                    $caseName,
                    $responseContent
                );
            } catch (Exception $e) {
                throw $e;
            }
        });
    }

    /**
     * Should fail when mandatory field (X-TIMESTAMP) is missing.
     */
    public function testApplyTokenFailMissingField(): void
    {
        Util::withDelay(function () {
            $caseName = 'ApplyTokenFailMissingField';
            $jsonDict = Util::getRequest(
                self::$jsonPathFile,
                self::$titleCase,
                $caseName
            );
            $jsonDict['authCode'] = 'test123';

            $headers = Util::getHeadersWithSignature(
                'POST',
                '/v1.0/access-token/b2b2c.htm',
                $jsonDict,
                false
            );
            $headers['X-TIMESTAMP'] = '';
            $headers['X-CLIENT-KEY'] = self::$partnerId;

            try {
                Util::executeApiRequest(
                    'POST',
                    'https://api.sandbox.dana.id/v1.0/access-token/b2b2c.htm',
                    $headers,
                    $jsonDict
                );
                $this->fail('Expected ApiException for missing field but the API call succeeded');
            } catch (ApiException $e) {
                $this->assertEquals(400, $e->getCode(), "Expected HTTP 400 Bad Request for missing field, got {$e->getCode()}");
                Assertion::assertFailResponse(
                    self::$jsonPathFile,
                    self::$titleCase,
                    $caseName,
                    (string)$e->getResponseBody()
                );
            } catch (Exception $e) {
                throw $e;
            }
        });
    }

    /**
     * Should fail when mandatory field format is invalid.
     */
    public function testApplyTokenFailInvalidField(): void
    {
        Util::withDelay(function () {
            $caseName = 'ApplyTokenFailInvalidField';
            $jsonDict = Util::getRequest(
                self::$jsonPathFile,
                self::$titleCase,
                $caseName
            );
            $jsonDict['authCode'] = 'test123';

            $headers = Util::getHeadersWithSignature(
                'POST',
                '/v1.0/access-token/b2b2c.htm',
                $jsonDict,
                true,
                true
            );
            $headers['X-CLIENT-KEY'] = self::$partnerId;

            try {
                Util::executeApiRequest(
                    'POST',
                    'https://api.sandbox.dana.id/v1.0/access-token/b2b2c.htm',
                    $headers,
                    $jsonDict
                );
                $this->fail('Expected ApiException for invalid field but the API call succeeded');
            } catch (ApiException $e) {
                $this->assertEquals(400, $e->getCode(), "Expected HTTP 400 Bad Request for invalid field, got {$e->getCode()}");
                Assertion::assertFailResponse(
                    self::$jsonPathFile,
                    self::$titleCase,
                    $caseName,
                    (string)$e->getResponseBody()
                );
            } catch (Exception $e) {
                throw $e;
            }
        });
    }
}
