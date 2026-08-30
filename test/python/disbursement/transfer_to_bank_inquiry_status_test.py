import os
import pytest
from dana.utils.snap_configuration import SnapConfiguration, AuthSettings, Env
from dana.disbursement.v1.models import (
    TransferToBankRequest,
    TransferToBankInquiryStatusRequest,
    TransferToBankInquiryStatusResponse,
)
from dana.disbursement.v1.api import DisbursementApi
from dana.api_client import ApiClient
from dana.exceptions import ApiException
from uuid import uuid4
from helper.util import get_request, with_delay, retry_on_inconsistent_request
from helper.assertion import assert_response

transfer_to_bank_title_case = "TransferToBank"
title_case = "TransferToBankInquiryStatus"
json_path_file = "resource/request/components/Disbursement.json"

configuration = SnapConfiguration(
    api_key=AuthSettings(
        PRIVATE_KEY=os.environ.get("PRIVATE_KEY"),
        ORIGIN=os.environ.get("ORIGIN"),
        X_PARTNER_ID=os.environ.get("X_PARTNER_ID"),
        DANA_ENV=Env.SANDBOX,
        X_DEBUG="true",
        CLIENT_SECRET=os.environ.get("CLIENT_SECRET"),
    )
)

with ApiClient(configuration) as api_client:
    api_instance = DisbursementApi(api_client)


def _create_transfer_to_bank_for_inquiry() -> str:
    case_name = "DisbursementBankValidAccount"
    json_dict = get_request(json_path_file, transfer_to_bank_title_case, case_name)
    partner_reference_no = str(uuid4())
    json_dict["partnerReferenceNo"] = partner_reference_no
    request_obj = TransferToBankRequest.from_dict(json_dict)
    api_instance.transfer_to_bank(request_obj)
    return partner_reference_no


@with_delay()
@retry_on_inconsistent_request(max_retries=3, delay_seconds=2)
def test_transfer_to_bank_inquiry_status_successful():
    original_partner_reference_no = _create_transfer_to_bank_for_inquiry()
    case_name = "TransferToBankInquiryStatusSuccessful"
    json_dict = get_request(json_path_file, title_case, case_name)
    json_dict["originalPartnerReferenceNo"] = original_partner_reference_no
    request_obj = TransferToBankInquiryStatusRequest.from_dict(json_dict)

    try:
        response = api_instance.transfer_to_bank_inquiry_status(request_obj)
        assert_response(
            json_path_file,
            title_case,
            case_name,
            TransferToBankInquiryStatusResponse.to_json(response),
            {"originalPartnerReferenceNo": original_partner_reference_no},
        )
    except ApiException as e:
        print(f"[REF] case={case_name} originalPartnerReferenceNo={original_partner_reference_no}")
        pytest.fail(f"API call failed: {e}")
