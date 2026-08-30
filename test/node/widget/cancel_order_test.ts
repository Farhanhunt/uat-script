import Dana, { ResponseError } from 'dana-node';
import { v4 as uuidv4 } from 'uuid';
import * as path from 'path';
import * as dotenv from 'dotenv';
import { fail } from 'assert';
import { getRequest, generateFormattedDate, retryTest } from '../helper/util';
import { assertResponse, assertFailResponse, assertSdkErrorResponse } from '../helper/assertion';
import { CancelOrderRequest, WidgetPaymentRequest } from 'dana-node/widget/v1';
import { executeManualApiRequest } from '../helper/apiHelpers';
import { createTestWidgetPaymentRefunded } from './payment_widget_util';

dotenv.config();

const titleCase = 'CancelOrder';
const jsonPathFile = path.resolve(__dirname, '../../../resource/request/components/Widget.json');
const baseUrl = 'https://api.sandbox.dana.id/';
const cancelApiPath = '/v1.0/debit/cancel.htm';
const merchantId = process.env.MERCHANT_ID || ''; // Merchant configuration
const userPhoneNumber = '083811223355';
const userPin = '181818';
const deviceId = 'deviceid123';

const dana = new Dana({
    partnerId: process.env.X_PARTNER_ID || '',
    privateKey: process.env.PRIVATE_KEY || '',
    origin: process.env.ORIGIN || '',
    env: process.env.ENV || 'sandbox',
});

function generateReferenceNo(): string {
    return uuidv4();
}

describe('CancelOrder Tests', () => {
    test('should cancel order with valid scenario', async () => {
        const caseName = 'CancelOrderValidScenario';

        const widgetPaymentRequestData: WidgetPaymentRequest = getRequest<WidgetPaymentRequest>(
            jsonPathFile,
            'Payment',
            'PaymentSuccess',
        );
        const partnerReferenceNo = generateReferenceNo();
        widgetPaymentRequestData.partnerReferenceNo = partnerReferenceNo;
        widgetPaymentRequestData.validUpTo = generateFormattedDate(900, 7);

        await dana.widgetApi.widgetPayment(widgetPaymentRequestData);
        await new Promise((resolve) => setTimeout(resolve, 2000));

        const cancelOrderRequestData: CancelOrderRequest = getRequest<CancelOrderRequest>(
            jsonPathFile,
            titleCase,
            caseName,
        );
        cancelOrderRequestData.originalPartnerReferenceNo = partnerReferenceNo;
        cancelOrderRequestData.merchantId = merchantId;

        const response = await dana.widgetApi.cancelOrder(cancelOrderRequestData);
        await assertResponse(jsonPathFile, titleCase, caseName, response, {
            partnerReferenceNo,
        });
    });

    test('should fail with user status abnormal', async () => {
        const caseName = 'CancelOrderFailUserStatusAbnormal';
        const requestData: CancelOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        try {
            const response = await dana.widgetApi.cancelOrder(requestData);
            await assertFailResponse(jsonPathFile, titleCase, caseName, response);
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('CancelOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with merchant status abnormal', async () => {
        const caseName = 'CancelOrderFailMerchantStatusAbnormal';
        const requestData: CancelOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        try {
            const response = await dana.widgetApi.cancelOrder(requestData);
            await assertFailResponse(jsonPathFile, titleCase, caseName, response);
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('CancelOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with missing parameter', async () => {
        const caseName = 'CancelOrderFailMissingParameter';
        const requestData: Record<string, unknown> = getRequest(jsonPathFile, titleCase, caseName);
        requestData.merchantId = merchantId;

        try {
            await executeManualApiRequest(
                caseName,
                'POST',
                baseUrl + cancelApiPath,
                cancelApiPath,
                requestData,
            );
            fail('Expected an error but the API call succeeded');
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else if (Number(e.status) === 400) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('CancelOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with order not exist', async () => {
        const caseName = 'CancelOrderFailOrderNotExist';

        await retryTest(3, 1000, async () => {
            const requestData: CancelOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
            requestData.originalPartnerReferenceNo = uuidv4();
            requestData.originalReferenceNo = uuidv4();
            const variableDict = { originalPartnerReferenceNo: requestData.originalPartnerReferenceNo };

            try {
                const response = await dana.widgetApi.cancelOrder(requestData);
                await assertSdkErrorResponse(jsonPathFile, titleCase, caseName, response, null, variableDict);
            } catch (e: any) {
                if (e instanceof ResponseError) {
                    await assertSdkErrorResponse(jsonPathFile, titleCase, caseName, null, e, variableDict);
                } else {
                    fail('CancelOrder test failed: ' + (e.message || e));
                }
            }
        });
    });

    test('should fail with exceed cancel window time', async () => {
        const caseName = 'CancelOrderFailExceedCancelWindowTime';
        const requestData: CancelOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        
        try {
            const response = await dana.widgetApi.cancelOrder(requestData);
            await assertFailResponse(jsonPathFile, titleCase, caseName, response);
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('CancelOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail not allowed by agreement', async () => {
        const caseName = 'CancelOrderFailNotAllowedByAgreement';
        const requestData: CancelOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        try {
            const response = await dana.widgetApi.cancelOrder(requestData);
            await assertFailResponse(jsonPathFile, titleCase, caseName, response);
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('CancelOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with account status abnormal', async () => {
        const caseName = 'CancelOrderFailAccountStatusAbnormal';
        const requestData: CancelOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        try {
            const response = await dana.widgetApi.cancelOrder(requestData);
            await assertFailResponse(jsonPathFile, titleCase, caseName, response);
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('CancelOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with insufficient merchant balance', async () => {
        const caseName = 'CancelOrderFailInsufficientMerchantBalance';
        const requestData: CancelOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        try {
            const response = await dana.widgetApi.cancelOrder(requestData);
            await assertFailResponse(jsonPathFile, titleCase, caseName, response);
        }
        catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('CancelOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with order invalid status', async () => {
        const caseName = 'CancelOrderFailOrderInvalidStatus';
        const partnerReferenceNo = await createTestWidgetPaymentRefunded();
        await new Promise((resolve) => setTimeout(resolve, 2000));

        const requestData: CancelOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        requestData.originalPartnerReferenceNo = partnerReferenceNo;
        requestData.merchantId = merchantId;

        try {
            await dana.widgetApi.cancelOrder(requestData);
            fail('Expected error but the API call succeeded');
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse), {
                    originalPartnerReferenceNo: partnerReferenceNo,
                });
            } else {
                fail('CancelOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with timeout', async () => {
        const caseName = 'CancelOrderFailTimeout';
        const requestData: CancelOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        try {
            const response = await dana.widgetApi.cancelOrder(requestData);
            await assertFailResponse(jsonPathFile, titleCase, caseName, response);
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('CancelOrder test failed: ' + (e.message || e));
            }
        }
    });
});
