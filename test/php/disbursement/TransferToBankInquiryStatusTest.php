<?php

namespace DanaUat\Disbursement;

use Dana\Disbursement\v1\Api\DisbursementApi as ApiDisbursementApi;
use Dana\ObjectSerializer;
use Dana\Configuration;
use Dana\Env;
use Dana\ApiException;
use DanaUat\Helper\Assertion;
use DanaUat\Helper\Util;
use Exception;

class TransferToBankInquiryStatusTest extends AbstractDisbursementTest
{
    private static $transferToBankTitleCase = 'TransferToBank';
    private static $titleCase = 'TransferToBankInquiryStatus';
    private static $jsonPathFile = 'resource/request/components/Disbursement.json';
    private static $apiInstance;

    public static function setUpBeforeClass(): void
    {
        parent::setUpBeforeClass();

        $configuration = new Configuration();
        $configuration->setApiKey('PRIVATE_KEY', getenv('PRIVATE_KEY'));
        $configuration->setApiKey('ORIGIN', getenv('ORIGIN'));
        $configuration->setApiKey('X_PARTNER_ID', getenv('X_PARTNER_ID'));
        $configuration->setApiKey('ENV', Env::SANDBOX);

        self::$apiInstance = new ApiDisbursementApi(null, $configuration);
    }

    protected function runTest(): void
    {
        Util::runWithRetry(
            function () {
                parent::runTest();
            },
            3,
            2000
        );
    }

    private function createTransferToBankForInquiry(): string
    {
        $caseName = 'DisbursementBankValidAccount';
        $jsonDict = Util::getRequest(
            self::$jsonPathFile,
            self::$transferToBankTitleCase,
            $caseName
        );
        $partnerReferenceNo = Util::generatePartnerReferenceNo();
        $jsonDict['partnerReferenceNo'] = $partnerReferenceNo;

        $transferToBankRequestObj = ObjectSerializer::deserialize(
            $jsonDict,
            'Dana\Disbursement\v1\Model\TransferToBankRequest'
        );

        self::$apiInstance->transferToBank($transferToBankRequestObj);

        return $partnerReferenceNo;
    }

    public function testTransferToBankInquiryStatusSuccessful(): void
    {
        Util::withDelay(function () {
            $originalPartnerReferenceNo = $this->createTransferToBankForInquiry();
            $caseName = 'TransferToBankInquiryStatusSuccessful';

            try {
                $jsonDict = Util::getRequest(
                    self::$jsonPathFile,
                    self::$titleCase,
                    $caseName
                );
                $jsonDict['originalPartnerReferenceNo'] = $originalPartnerReferenceNo;

                $inquiryRequestObj = ObjectSerializer::deserialize(
                    $jsonDict,
                    'Dana\Disbursement\v1\Model\TransferToBankInquiryStatusRequest'
                );

                $apiResponse = self::$apiInstance->transferToBankInquiryStatus($inquiryRequestObj);

                Assertion::assertResponse(
                    self::$jsonPathFile,
                    self::$titleCase,
                    $caseName,
                    $apiResponse->__toString(),
                    ['originalPartnerReferenceNo' => $originalPartnerReferenceNo]
                );
                $this->assertTrue(true);
            } catch (ApiException $e) {
                $this->fail(
                    '[REF] case=' . $caseName
                    . ' originalPartnerReferenceNo=' . $originalPartnerReferenceNo
                    . ' error=' . $e->getMessage()
                );
            } catch (Exception $e) {
                $this->fail('Unexpected exception: ' . $e->getMessage());
            }
        });
    }
}
