import Dana, { ResponseError } from 'dana-node';
import { v4 as uuidv4 } from 'uuid';
import * as path from 'path';
import * as dotenv from 'dotenv';
import { fail } from 'assert';
import { getRequest } from '../helper/util';
import { assertResponse, assertFailResponse } from '../helper/assertion';
import { RefundOrderRequest } from 'dana-node/widget/v1';
import { executeManualApiRequest } from '../helper/apiHelpers';
import {
    createTestWidgetPaymentInit,
    createTestWidgetPaymentPaid,
} from './payment_widget_util';

dotenv.config();

const titleCase = 'RefundOrder';
const jsonPathFile = path.resolve(__dirname, '../../../resource/request/components/Widget.json');
const baseUrl = 'https://api.sandbox.dana.id/';
const apiPath = '/v1.0/debit/refund.htm';
const merchantId = process.env.MERCHANT_ID || '';

const dana = new Dana({
    partnerId: process.env.X_PARTNER_ID || '',
    privateKey: process.env.PRIVATE_KEY || '',
    origin: process.env.ORIGIN || '',
    env: process.env.ENV || 'sandbox',
});

function generateReferenceNo(): string {
    return uuidv4();
}

describe('RefundOrder Tests', () => {
    test('should successfully refund order (valid scenario)', async () => {
        const caseName = 'RefundOrderValidScenario';
        const partnerReferenceNo = await createTestWidgetPaymentPaid();
        await new Promise((resolve) => setTimeout(resolve, 2000));

        const requestData = getRequest<RefundOrderRequest>(jsonPathFile, titleCase, caseName);
        requestData.originalPartnerReferenceNo = partnerReferenceNo;
        requestData.partnerRefundNo = partnerReferenceNo;
        requestData.merchantId = merchantId;

        const response = await dana.widgetApi.refundOrder(requestData);
        await assertResponse(jsonPathFile, titleCase, caseName, JSON.stringify(response), {
            partnerReferenceNo,
        });
    });

    test.skip('should successfully refund order (in process)', async () => {
        const caseName = 'RefundInProcess';
        const requestData: RefundOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        requestData.partnerRefundNo = generateReferenceNo();

        try {
            const response = await dana.widgetApi.refundOrder(requestData);
            await assertResponse(jsonPathFile, titleCase, caseName, JSON.stringify(response));
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('RefundOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with duplicate request', async () => {
        const caseName = 'RefundFailDuplicateRequest';
        const partnerReferenceNo = await createTestWidgetPaymentPaid();
        await new Promise((resolve) => setTimeout(resolve, 2000));

        const requestData = getRequest<RefundOrderRequest>(jsonPathFile, titleCase, caseName);
        requestData.originalPartnerReferenceNo = partnerReferenceNo;
        requestData.partnerRefundNo = partnerReferenceNo;
        requestData.merchantId = merchantId;

        await dana.widgetApi.refundOrder(requestData);
        await new Promise((resolve) => setTimeout(resolve, 2000));

        requestData.refundAmount = { currency: 'IDR', value: '2.00' };

        try {
            await dana.widgetApi.refundOrder(requestData);
            fail('Expected duplicate refund error but API call succeeded');
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse), {
                    partnerReferenceNo,
                });
            } else {
                fail('RefundOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with order not paid', async () => {
        const caseName = 'RefundFailOrderNotPaid';
        const partnerReferenceNo = await createTestWidgetPaymentInit();
        await new Promise((resolve) => setTimeout(resolve, 2000));

        const requestData = getRequest<RefundOrderRequest>(jsonPathFile, titleCase, caseName);
        requestData.originalPartnerReferenceNo = partnerReferenceNo;
        requestData.partnerRefundNo = partnerReferenceNo;
        requestData.merchantId = merchantId;

        try {
            await dana.widgetApi.refundOrder(requestData);
            fail('Expected error but the API call succeeded');
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse), {
                    originalPartnerReferenceNo: partnerReferenceNo,
                });
            } else {
                fail('RefundOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with mandatory parameter invalid', async () => {
        const caseName = 'RefundFailMandatoryParameterInvalid';
        const requestData: RefundOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        const customerHeaders: Record<string, string> = {
            'X-TIMESTAMP': '',
        };

        try {
            await executeManualApiRequest(
                caseName,
                'POST',
                baseUrl + apiPath,
                apiPath,
                requestData,
                customerHeaders,
            );
            fail('Expected an error but the API call succeeded');
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else if (Number(e.status) === 400) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('RefundOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with order not exist', async () => {
        const caseName = 'RefundFailOrderNotExist';
        const requestData: RefundOrderRequest = getRequest(jsonPathFile, titleCase, caseName);

        try {
            const response = await dana.widgetApi.refundOrder(requestData);
            await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(response));
            fail('Expected an error but the API call succeeded');
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('RefundOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with invalid signature', async () => {
        const caseName = 'RefundFailInvalidSignature';
        const requestData: RefundOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        const customerHeaders: Record<string, string> = {
            'X-SIGNATURE': 'invalid_signature',
        };

        try {
            await executeManualApiRequest(
                caseName,
                'POST',
                baseUrl + apiPath,
                apiPath,
                requestData,
                customerHeaders,
            );
            fail('Expected an error but the API call succeeded');
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else if (Number(e.status) === 401) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('RefundOrder test failed: ' + (e.message || e));
            }
        }
    });

    test('should fail with timeout', async () => {
        const caseName = 'RefundFailTimeout';
        const requestData: RefundOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        requestData.partnerRefundNo = generateReferenceNo();

        try {
            const response = await dana.widgetApi.refundOrder(requestData);
            await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(response));
            fail('Expected an error but the API call succeeded');
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('RefundOrder test failed: ' + (e.message || e));
            }
        }
    });

    test.skip('should fail with idempotent', async () => {
        const caseName = 'RefundFailIdempotent';
        const requestData: RefundOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        fail('RefundOrder idempotent test skipped (same as Go)');
    });

    test('should fail with merchant status abnormal', async () => {
        const caseName = 'RefundFailMerchantStatusAbnormal';
        const requestData: RefundOrderRequest = getRequest(jsonPathFile, titleCase, caseName);
        requestData.partnerRefundNo = generateReferenceNo();

        try {
            const response = await dana.widgetApi.refundOrder(requestData);
            await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(response));
            fail('Expected an error but the API call succeeded');
        } catch (e: any) {
            if (e instanceof ResponseError) {
                await assertFailResponse(jsonPathFile, titleCase, caseName, JSON.stringify(e.rawResponse));
            } else {
                fail('RefundOrder test failed: ' + (e.message || e));
            }
        }
    });
});
