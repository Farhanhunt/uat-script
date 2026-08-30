import Dana from 'dana-node';
import { v4 as uuidv4 } from 'uuid';
import * as path from 'path';
import * as dotenv from 'dotenv';
import {
    automatePayment,
    generateFormattedDate,
    getRequest,
    retryOnInconsistentRequest,
} from '../helper/util';
import { RefundOrderRequest, WidgetPaymentRequest } from 'dana-node/widget/v1';

dotenv.config();

const jsonPathFile = path.resolve(__dirname, '../../../resource/request/components/Widget.json');
const merchantId = process.env.MERCHANT_ID || '';
const userPhoneNumber = '083811223355';
const userPin = '181818';

const dana = new Dana({
    partnerId: process.env.X_PARTNER_ID || '',
    privateKey: process.env.PRIVATE_KEY || '',
    origin: process.env.ORIGIN || '',
    env: process.env.ENV || 'sandbox',
});

export async function createTestWidgetPaymentInit(): Promise<string> {
    return retryOnInconsistentRequest(async () => {
        const requestData = getRequest<WidgetPaymentRequest>(jsonPathFile, 'Payment', 'PaymentSuccess');
        const partnerReferenceNo = uuidv4();
        requestData.partnerReferenceNo = partnerReferenceNo;
        requestData.validUpTo = generateFormattedDate(900, 7);
        requestData.merchantId = merchantId;
        await dana.widgetApi.widgetPayment(requestData);
        return partnerReferenceNo;
    });
}

export async function createTestWidgetPaymentWithRedirect(): Promise<{
    partnerReferenceNo: string;
    webRedirectUrl: string;
}> {
    return retryOnInconsistentRequest(async () => {
        const requestData = getRequest<WidgetPaymentRequest>(jsonPathFile, 'Payment', 'PaymentSuccess');
        const partnerReferenceNo = uuidv4();
        requestData.partnerReferenceNo = partnerReferenceNo;
        requestData.validUpTo = generateFormattedDate(900, 7);
        requestData.merchantId = merchantId;

        const response = await dana.widgetApi.widgetPayment(requestData);
        if (!response.webRedirectUrl) {
            throw new Error('No webRedirectUrl in widget payment response');
        }

        return { partnerReferenceNo, webRedirectUrl: response.webRedirectUrl };
    });
}

export async function createTestWidgetPaymentPaid(): Promise<string> {
    const { partnerReferenceNo, webRedirectUrl } = await createTestWidgetPaymentWithRedirect();

    const automationResult = await automatePayment(
        userPhoneNumber,
        userPin,
        webRedirectUrl,
        3,
        2000,
        true,
    );

    if (!automationResult.success) {
        throw new Error(`Payment automation failed: ${automationResult.error}`);
    }

    await new Promise((resolve) => setTimeout(resolve, 5000));
    return partnerReferenceNo;
}

export async function createTestWidgetPaymentRefunded(): Promise<string> {
    return retryOnInconsistentRequest(async () => {
        const partnerReferenceNo = await createTestWidgetPaymentPaid();

        const refundRequest = getRequest<RefundOrderRequest>(
            jsonPathFile,
            'RefundOrder',
            'RefundOrderValidScenario',
        );
        refundRequest.originalPartnerReferenceNo = partnerReferenceNo;
        refundRequest.partnerRefundNo = partnerReferenceNo;
        refundRequest.merchantId = merchantId;

        await dana.widgetApi.refundOrder(refundRequest);
        return partnerReferenceNo;
    });
}
