package id.dana.disbursement;

import id.dana.disbursement.v1.api.DisbursementApi;
import id.dana.disbursement.v1.model.TransferToBankInquiryStatusRequest;
import id.dana.disbursement.v1.model.TransferToBankInquiryStatusResponse;
import id.dana.disbursement.v1.model.TransferToBankRequest;
import id.dana.disbursement.v1.model.TransferToBankResponse;
import id.dana.invoker.Dana;
import id.dana.invoker.model.DanaConfig;
import id.dana.invoker.model.constant.EnvKey;
import id.dana.invoker.model.enumeration.DanaEnvironment;
import id.dana.util.ConfigUtil;
import id.dana.util.RetryTestUtil;
import id.dana.util.TestUtil;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;
import java.util.UUID;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

class TransferToBankInquiryStatusTest extends AbstractDisbursementTest {

  private static final Logger log = LoggerFactory.getLogger(TransferToBankInquiryStatusTest.class);

  private static final String transferToBankTitleCase = "TransferToBank";
  private static final String titleCase = "TransferToBankInquiryStatus";
  private static final String jsonPathFile = TransferToBankInquiryStatusTest.class.getResource(
      "/request/components/Disbursement.json").getPath();

  private DisbursementApi api;

  @BeforeEach
  void setUp() {
    DanaConfig.Builder danaConfigBuilder = new DanaConfig.Builder();
    danaConfigBuilder
        .partnerId(ConfigUtil.getConfig("X_PARTNER_ID", ""))
        .privateKey(ConfigUtil.getConfig("PRIVATE_KEY", ""))
        .origin(ConfigUtil.getConfig("ORIGIN", ""))
        .env(DanaEnvironment.getByName(ConfigUtil.getConfig(EnvKey.ENV, "SANDBOX")));

    DanaConfig.getInstance(danaConfigBuilder);
    api = Dana.getInstance().getDisbursementApi();
  }

  private String createTransferToBankForInquiry() throws IOException {
    String caseName = "DisbursementBankValidAccount";
    TransferToBankRequest requestData = TestUtil.getRequest(
        jsonPathFile, transferToBankTitleCase, caseName, TransferToBankRequest.class);

    String partnerReferenceNo = UUID.randomUUID().toString();
    log.info("[REF] case={} partnerReferenceNo={}", caseName, partnerReferenceNo);
    requestData.setPartnerReferenceNo(partnerReferenceNo);

    TransferToBankResponse response = api.transferToBank(requestData);
    if (response.getPartnerReferenceNo() != null && !response.getPartnerReferenceNo().isEmpty()) {
      return response.getPartnerReferenceNo();
    }
    return partnerReferenceNo;
  }

  @Test
  @RetryTestUtil.Retry(value = 3, waitMs = 2000)
  void testTransferToBankInquiryStatusSuccessful() throws IOException {
    String originalPartnerReferenceNo = createTransferToBankForInquiry();
    String caseName = "TransferToBankInquiryStatusSuccessful";

    TransferToBankInquiryStatusRequest requestData = TestUtil.getRequest(
        jsonPathFile, titleCase, caseName, TransferToBankInquiryStatusRequest.class);
    requestData.setOriginalPartnerReferenceNo(originalPartnerReferenceNo);

    Map<String, Object> variableDict = new HashMap<>();
    variableDict.put("originalPartnerReferenceNo", originalPartnerReferenceNo);

    TransferToBankInquiryStatusResponse response = api.transferToBankInquiryStatus(requestData);
    TestUtil.assertResponse(jsonPathFile, titleCase, caseName, response, variableDict);
  }
}
