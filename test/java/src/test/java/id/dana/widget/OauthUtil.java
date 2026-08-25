package id.dana.widget;

import id.dana.invoker.model.exception.DanaException;
import id.dana.util.BrowserTestSupport;
import id.dana.util.ConfigUtil;
import id.dana.util.TestUtil;
import id.dana.widget.v1.model.Oauth2UrlData;
import id.dana.widget.v1.model.Oauth2UrlDataSeamlessData;
import id.dana.widget.v1.util.WidgetUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.UnsupportedEncodingException;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.security.*;
import java.security.spec.InvalidKeySpecException;
import java.security.spec.PKCS8EncodedKeySpec;
import java.util.Base64;
import java.util.regex.Pattern;

public class OauthUtil {
    private static String authCode;
    private static final Logger log = LoggerFactory.getLogger(TestUtil.class);
    private static final String redirectUrl = ConfigUtil.getConfig("REDIRECT_URL_OAUTH", "https://google.com");
    private static final Pattern NON_DIGITS = Pattern.compile("\\D+");

    public static String generateSeamlessData(
            String phoneNumber,
            String bizScenario,
            String timeVerified,
            String externalUid,
            String deviceId,
            Boolean skipRegisterConsult)
            throws UnsupportedEncodingException {

        if (skipRegisterConsult == null) {
            skipRegisterConsult = true;
        }

        return String.format(
                "{\"phoneNumber\":\"%s\",\"bizScenario\":\"%s\",\"timeVerified\":\"%s\",\"externalUid\":\"%s\",\"deviceId\":\"%s\",\"skipRegisterConsult\":%b}",
                phoneNumber, bizScenario, timeVerified, externalUid, deviceId, skipRegisterConsult);
    }

    public static String generateSeamlessSign(
            String seamlessData) throws
            UnsupportedEncodingException,
            NoSuchAlgorithmException,
            InvalidKeySpecException,
            SignatureException,
            InvalidKeyException {

        String privateKey = ConfigUtil.getConfig("PRIVATE_KEY", "");
        String signResult = sign(seamlessData, privateKey);

        return URLEncoder.encode(signResult, String.valueOf(StandardCharsets.UTF_8));
    }

    private static String sign(String textPayload, String privateKeyMerchant)
            throws NoSuchAlgorithmException,
            InvalidKeySpecException,
            SignatureException,
            InvalidKeyException {

        PrivateKey privateKeyObject = getPrivateKey(privateKeyMerchant);
        Signature signatureProcessor = Signature.getInstance("SHA256withRSA");
        signatureProcessor.initSign(privateKeyObject);
        signatureProcessor.update(textPayload.getBytes());

        byte[] signature = signatureProcessor.sign();

        return new String(Base64.getEncoder().encode(signature));
    }

    public static PrivateKey getPrivateKey(String privateKeyMerchant)
            throws NoSuchAlgorithmException,
            InvalidKeySpecException {

        String base64 = privateKeyMerchant
                .replace("\\n", "")
                .replace("\\r", "")
                .replaceAll("-----BEGIN PRIVATE KEY-----", "")
                .replaceAll("-----END PRIVATE KEY-----", "")
                .replaceAll("-----BEGIN RSA PRIVATE KEY-----", "")
                .replaceAll("-----END RSA PRIVATE KEY-----", "")
                .replaceAll("\\s+", "");
        byte[] key = Base64.getDecoder().decode(base64);
        PKCS8EncodedKeySpec pkcs8EncodedKeySpec = new PKCS8EncodedKeySpec(key);
        KeyFactory keyFactory = KeyFactory.getInstance("RSA");

        return keyFactory.generatePrivate(pkcs8EncodedKeySpec);
    }

    /** @deprecated Legacy manual URL builder; prefer {@link #getRedirectOauthUrl(String)}. */
    @Deprecated
    public static String generateRedirectLinkAuthCode(
            String partnerId,
            String channelId,
            String scope,
            String redirectUrlParam,
            String seamlessData,
            String seamlessSign) throws UnsupportedEncodingException {

        String basePath = "https://m.sandbox.dana.id/";
        String path = "v1.0/get-auth-code";

        String encodedSeamlessData = URLEncoder.encode(seamlessData, StandardCharsets.UTF_8.toString());
        String encodedSeamlessSign = URLEncoder.encode(seamlessSign, StandardCharsets.UTF_8.toString());

        return basePath + path + "?" +
                "partnerId=" + partnerId +
                "&timestamp=2023-08-31T22:27:48+00:00" +
                "&externalId=test" +
                "&channelId=" + channelId +
                "&scopes=" + scope +
                "&redirectUrl=" + redirectUrlParam +
                "&state=22321" +
                "&seamlessData=" + encodedSeamlessData +
                "&seamlessSign=" + encodedSeamlessSign;
    }

    /**
     * Builds OAuth URL via SDK ({@link WidgetUtil#generateOauthUrl}) with minimal seamlessData —
     * mobile only — matching Go {@code GetRedirectOauthUrl}.
     */
    public static String getRedirectOauthUrl(String phoneNumber) {
        String normalizedPhone = normalizeMobileNumber(phoneNumber);

        Oauth2UrlData oauth2UrlData = new Oauth2UrlData();
        oauth2UrlData.setRedirectUrl(redirectUrl);
        oauth2UrlData.setMerchantId(ConfigUtil.getConfig("MERCHANT_ID", ""));

        Oauth2UrlDataSeamlessData seamlessData = new Oauth2UrlDataSeamlessData();
        seamlessData.setMobileNumber(normalizedPhone);
        oauth2UrlData.setSeamlessData(seamlessData);

        try {
            String oauthUrl = WidgetUtil.generateOauthUrl(oauth2UrlData);
            log.info("RedirectOauthUrl: {}", oauthUrl);
            return oauthUrl;
        } catch (DanaException e) {
            throw new RuntimeException("Failed to generate OAuth URL", e);
        }
    }

    public static String getAuthCode(
            String partnerId,
            String channelId,
            String phoneNumberUser,
            String pinUser) {

        String urlRedirectAuth = getRedirectOauthUrl(phoneNumberUser);
        return getOauthViaView(urlRedirectAuth, phoneNumberUser, pinUser);
    }

    public static String getOauthViaView(String urlRedirectLinkAuthCode, String phoneNumber, String pin) {
        return BrowserTestSupport.oauthGetOauthViaView(urlRedirectLinkAuthCode, phoneNumber, pin);
    }

    public static String getAccessToken(String phoneNumberUser, String pinUser) throws
            UnsupportedEncodingException,
            NoSuchAlgorithmException,
            InvalidKeySpecException,
            SignatureException,
            InvalidKeyException {

        authCode = getAuthCode(
                ConfigUtil.getConfig("X_PARTNER_ID", ""),
                ConfigUtil.getConfig("X_PARTNER_ID", ""),
                phoneNumberUser,
                pinUser);

        return ApplyToken.applyToken(authCode);
    }

    static String normalizeMobileNumber(String mobile) {
        if (mobile == null || mobile.trim().isEmpty()) {
            return "083811223355";
        }
        String digits = NON_DIGITS.matcher(mobile).replaceAll("");
        if (digits.isEmpty()) {
            return "083811223355";
        }
        if (digits.startsWith("62")) {
            return "0" + digits.substring(2);
        }
        return digits;
    }
}
