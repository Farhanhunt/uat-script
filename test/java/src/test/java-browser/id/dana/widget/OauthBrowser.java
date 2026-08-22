package id.dana.widget;

import com.microsoft.playwright.Browser;
import com.microsoft.playwright.BrowserType;
import com.microsoft.playwright.Locator;
import com.microsoft.playwright.Page;
import com.microsoft.playwright.Playwright;
import id.dana.util.ConfigUtil;
import id.dana.util.TestUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Playwright-based OAuth browser flow. Built only when the {@code with-playwright} Maven profile is active.
 */
public final class OauthBrowser {

    private static final Logger log = LoggerFactory.getLogger(TestUtil.class);
    private static final String REDIRECT_URL_OAUTH = ConfigUtil.getConfig("REDIRECT_URL_OAUTH", "https://google.com");
    private static final int MAX_OAUTH_RETRIES = 3;
    private static final long RETRY_DELAY_MS = 3000;
    private static final int AUTH_CODE_TIMEOUT_MS = 45000;
    private static final int NAVIGATION_TIMEOUT_MS = 60000;
    private static final int LOCATOR_TIMEOUT_MS = 15000;
    private static final Pattern AUTH_CODE_PATTERN = Pattern.compile(
            "(?:^|[?&])(?:authCode|auth_code|auth-code)=([^&]+)");

    private OauthBrowser() {
    }

    public static String getOauthViaView(String urlRedirectLinkAuthCode, String phoneNumber, String pin) {
        String normalizedPhone = normalizePhone(phoneNumber);
        RuntimeException lastError = null;

        for (int attempt = 1; attempt <= MAX_OAUTH_RETRIES; attempt++) {
            log.info("OAuth attempt {}/{} (phone={})", attempt, MAX_OAUTH_RETRIES, normalizedPhone);
            try {
                String authCode = runOauthOnce(urlRedirectLinkAuthCode, normalizedPhone, pin);
                if (authCode != null && !authCode.isBlank()) {
                    log.info("Auth Code: {}", authCode);
                    return authCode;
                }
                lastError = new RuntimeException("empty auth code");
            } catch (RuntimeException e) {
                lastError = e;
                log.warn("OAuth attempt {}/{} failed: {}", attempt, MAX_OAUTH_RETRIES, e.getMessage());
            }

            if (attempt < MAX_OAUTH_RETRIES) {
                sleepQuietly(RETRY_DELAY_MS);
            }
        }

        throw lastError != null
                ? lastError
                : new RuntimeException("OAuth failed after " + MAX_OAUTH_RETRIES + " attempts");
    }

    private static String runOauthOnce(String urlRedirectLinkAuthCode, String phoneNumber, String pin) {
        try (Playwright playwright = Playwright.create()) {
            BrowserType.LaunchOptions opts = new BrowserType.LaunchOptions()
                    .setHeadless(true)
                    .setArgs(java.util.Arrays.asList(
                            "--disable-web-security",
                            "--disable-features=IsolateOrigins",
                            "--disable-site-isolation-trials",
                            "--disable-blink-features=AutomationControlled",
                            "--no-sandbox",
                            "--disable-dev-shm-usage"));
            Browser browser = playwright.chromium().launch(opts);
            try {
                Page page = browser.newPage();
                page.setDefaultTimeout(LOCATOR_TIMEOUT_MS);
                page.navigate(urlRedirectLinkAuthCode, new Page.NavigateOptions()
                        .setTimeout(NAVIGATION_TIMEOUT_MS)
                        .setWaitUntil(com.microsoft.playwright.options.WaitUntilState.DOMCONTENTLOADED));

                sleepQuietly(2000);

                String inputPhoneNumber = ".desktop-input>.txt-input-phone-number-field, input.txt-input-phone-number-field, input[type='tel']";
                String buttonSubmitPhoneNumber = ".agreement__button>.btn-continue, button.btn-continue";
                String inputPin = ".txt-input-pin-field, input[maxlength='6'][inputmode='numeric'], input[type='password']";

                Locator pinField = page.locator(inputPin).first();
                if (!pinField.isVisible()) {
                    Locator phoneField = page.locator(inputPhoneNumber).first();
                    phoneField.waitFor(new Locator.WaitForOptions().setTimeout(LOCATOR_TIMEOUT_MS));
                    if (phoneField.isVisible()) {
                        String current = phoneField.inputValue();
                        if (current == null || current.isBlank()) {
                            phoneField.fill(phoneNumber);
                        }
                    }
                    page.locator(buttonSubmitPhoneNumber).first().click();
                    pinField.waitFor(new Locator.WaitForOptions().setTimeout(LOCATOR_TIMEOUT_MS));
                }

                pinField.fill(pin);

                Locator continueAfterPin = page.locator(buttonSubmitPhoneNumber).first();
                if (continueAfterPin.isVisible()) {
                    continueAfterPin.click();
                }

                sleepQuietly(1500);

                return waitForAuthCode(page);
            } finally {
                browser.close();
            }
        }
    }

    private static String waitForAuthCode(Page page) {
        long deadline = System.currentTimeMillis() + AUTH_CODE_TIMEOUT_MS;
        String lastUrl = "";

        while (System.currentTimeMillis() < deadline) {
            String currentUrl = page.url();
            if (!currentUrl.equals(lastUrl)) {
                log.info("OAuth URL: {}", currentUrl);
                lastUrl = currentUrl;
            }

            String authCode = extractAuthCodeFromUrl(currentUrl);
            if (authCode != null && !authCode.isBlank()) {
                return authCode;
            }

            sleepQuietly(500);
        }

        throw new RuntimeException("Timeout " + AUTH_CODE_TIMEOUT_MS + "ms exceeded waiting for authCode redirect");
    }

    static String extractAuthCodeFromUrl(String url) {
        if (url == null || url.isBlank()) {
            return null;
        }

        Matcher matcher = AUTH_CODE_PATTERN.matcher(url);
        if (matcher.find()) {
            return URLDecoder.decode(matcher.group(1), StandardCharsets.UTF_8);
        }

        if (url.contains("authCode=")) {
            String fragment = url.split("authCode=", 2)[1];
            return URLDecoder.decode(fragment.split("&")[0], StandardCharsets.UTF_8);
        }

        return null;
    }

    private static String normalizePhone(String phoneNumber) {
        if (phoneNumber == null || phoneNumber.isBlank()) {
            return phoneNumber;
        }
        return phoneNumber.startsWith("0") ? phoneNumber.substring(1) : phoneNumber;
    }

    private static void sleepQuietly(long ms) {
        try {
            Thread.sleep(ms);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new RuntimeException(e);
        }
    }
}
