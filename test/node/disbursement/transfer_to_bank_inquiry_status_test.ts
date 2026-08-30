import Dana from 'dana-node';
import { v4 as uuidv4 } from 'uuid';
import * as path from 'path';
import * as dotenv from 'dotenv';
import { getRequest, retryOnInconsistentRequest } from '../helper/util';
import { assertResponse } from '../helper/assertion';

dotenv.config();

const transferToBankTitleCase = 'TransferToBank';
const titleCase = 'TransferToBankInquiryStatus';
const jsonPathFile = path.resolve(__dirname, '../../../resource/request/components/Disbursement.json');

const dana = new Dana({
  partnerId: process.env.X_PARTNER_ID || '',
  privateKey: process.env.PRIVATE_KEY || '',
  origin: process.env.ORIGIN || '',
  env: process.env.ENV || 'sandbox',
});

async function createTransferToBankForInquiry(): Promise<string> {
  let partnerReferenceNo = '';
  await retryOnInconsistentRequest(async () => {
    partnerReferenceNo = uuidv4();
    const requestData: any = getRequest(jsonPathFile, transferToBankTitleCase, 'DisbursementBankValidAccount');
    requestData.partnerReferenceNo = partnerReferenceNo;
    await dana.disbursementApi.transferToBank(requestData);
  }, 3, 2000);
  return partnerReferenceNo;
}

describe('Disbursement - Transfer To Bank Inquiry Status Tests', () => {
  jest.retryTimes(2, { logErrorsBeforeRetry: true });

  test('TransferToBankInquiryStatusSuccessful - should successfully inquire transfer to bank status', async () => {
    const originalPartnerReferenceNo = await createTransferToBankForInquiry();
    const caseName = 'TransferToBankInquiryStatusSuccessful';
    const requestData: any = getRequest(jsonPathFile, titleCase, caseName);
    requestData.originalPartnerReferenceNo = originalPartnerReferenceNo;

    try {
      const response = await dana.disbursementApi.transferToBankInquiryStatus(requestData);
      await assertResponse(jsonPathFile, titleCase, caseName, response, {
        originalPartnerReferenceNo,
      });
    } catch (e) {
      console.error(`[REF] case=${caseName} originalPartnerReferenceNo=${originalPartnerReferenceNo}`);
      throw e;
    }
  });
});
