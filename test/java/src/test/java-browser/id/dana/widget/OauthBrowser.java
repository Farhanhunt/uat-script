package id.dana.widget;

import com.microsoft.playwright.Browser;
import com.microsoft.playwright.BrowserContext;
import com.microsoft.playwright.BrowserType;
import com.microsoft.playwright.Keyboard;
import com.microsoft.playwright.Locator;
import com.microsoft.playwright.Page;
import com.microsoft.playwright.Playwright;
import com.microsoft.playwright.options.Geolocation;
import id.dana.util.TestUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.UnsupportedEncodingException;
import java.net.URI;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.concurrent.atomic.AtomicReference;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Playwright-based OAuth browser flow aligned with Go
 * {@code test/go/widget/oauth_widget_util.go} / {@code widget_oauth_automation.go}.
 * Built only when the {@code with-playwright} Maven profile is active.
 */
public final class OauthBrowser {

    private static final Logger log = LoggerFactory.getLogger(TestUtil.class);
    private static final int MAX_OAUTH_RETRIES = 3;
    private static final long RETRY_DELAY_MS = 3000;
    /** Matches Go AutomateOAuth auth-code wait (30s). */
    private static final int AUTH_CODE_TIMEOUT_MS = 30000;
    private static final int NAVIGATION_TIMEOUT_MS = 60000;
    private static final int LOCATOR_TIMEOUT_MS = 15000;
    private static final Pattern AUTH_CODE_PATTERN = Pattern.compile(
            "(?:^|[?&])(?:authCode|auth_code|auth-code)=([^&]+)");
    private static final String PIN_SELECTOR =
            ".txt-input-pin-field, input[maxlength='6'][inputmode='numeric'], input[type='password']";

    private OauthBrowser() {
    }

    public static String getOauthViaView(String urlRedirectLinkAuthCode, String phoneNumber, String pin) {
        String normalizedPhone = OauthUtil.normalizeMobileNumber(phoneNumber);
        if (isBlank(pin)) {
            pin = "181818";
        }
        RuntimeException lastError = null;

        for (int attempt = 1; attempt <= MAX_OAUTH_RETRIES; attempt++) {
            log.info("OAuth attempt {}/{} (phone={})", attempt, MAX_OAUTH_RETRIES, normalizedPhone);
            try {
                // Attempt 1 may use caller URL; later attempts regenerate (fresh externalId/timestamp).
                String oauthUrl = (attempt == 1 && !isBlank(urlRedirectLinkAuthCode))
                        ? urlRedirectLinkAuthCode
                        : OauthUtil.getRedirectOauthUrl(normalizedPhone);
                String authCode = runOauthOnce(oauthUrl, normalizedPhone, pin);
                if (!isBlank(authCode)) {
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
            BrowserType.LaunchOptions launchOptions = new BrowserType.LaunchOptions()
                    .setHeadless(true)
                    .setArgs(Arrays.asList(
                            "--disable-web-security",
                            "--disable-features=IsolateOrigins",
                            "--disable-site-isolation-trials",
                            "--disable-blink-features=AutomationControlled",
                            "--no-sandbox",
                            "--disable-dev-shm-usage"));
            Browser browser = playwright.chromium().launch(launchOptions);
            try {
                // iPhone 13 device profile (Playwright Java has no devices() map like Go).
                Browser.NewContextOptions contextOptions = new Browser.NewContextOptions()
                        .setViewportSize(390, 844)
                        .setDeviceScaleFactor(3)
                        .setIsMobile(true)
                        .setHasTouch(true)
                        .setLocale("id-ID")
                        .setUserAgent("Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) "
                                + "AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Mobile/15E148 Safari/604.1");
                contextOptions.setGeolocation(new Geolocation(-6.2088, 106.8456));
                contextOptions.setPermissions(Arrays.asList("geolocation"));

                BrowserContext context = browser.newContext(contextOptions);
                try {
                    context.setDefaultTimeout(NAVIGATION_TIMEOUT_MS);
                    Page page = context.newPage();
                    // Capture authCode as soon as any frame hits it — Google often
                    // strips ?authCode= from the URL on the next redirect.
                    AtomicReference<String> capturedAuthCode = new AtomicReference<>();
                    registerAuthCodeCapture(page, capturedAuthCode);
                    registerChromeErrorHandler(page, capturedAuthCode);
                    navigateToOAuthUrl(page, urlRedirectLinkAuthCode);
                    sleepQuietly(2000);

                    Locator pinField = page.locator(PIN_SELECTOR).first();
                    if (!pinField.isVisible(new Locator.IsVisibleOptions().setTimeout(0))) {
                        fillPhoneNumber(page, phoneNumber);
                        sleepQuietly(1000);
                        clickContinueAfterPhone(page);
                        sleepQuietly(3000);
                        clickOptionalContinue(page);
                        try {
                            pinField.waitFor(new Locator.WaitForOptions()
                                    .setState(com.microsoft.playwright.options.WaitForSelectorState.ATTACHED)
                                    .setTimeout(LOCATOR_TIMEOUT_MS));
                        } catch (RuntimeException e) {
                            log.warn("Timeout waiting for PIN field after phone submit");
                        }
                    }

                    enterPin(page, pin);
                    return waitForAuthCode(page, capturedAuthCode);
                } finally {
                    context.close();
                }
            } finally {
                browser.close();
            }
        }
    }

    private static void registerAuthCodeCapture(Page page, AtomicReference<String> capturedAuthCode) {
        page.onFrameNavigated(frame -> {
            String authCode = extractAuthCodeFromRedirect(frame.url());
            if (!isBlank(authCode) && capturedAuthCode.compareAndSet(null, authCode)) {
                log.info("Captured authCode from navigation: {}", authCode);
            }
        });
        page.onRequest(request -> {
            String authCode = extractAuthCodeFromRedirect(request.url());
            if (!isBlank(authCode) && capturedAuthCode.compareAndSet(null, authCode)) {
                log.info("Captured authCode from request: {}", authCode);
            }
        });
    }

    private static void registerChromeErrorHandler(Page page, AtomicReference<String> capturedAuthCode) {
        page.onFrameNavigated(frame -> {
            if (!isBlank(capturedAuthCode.get())) {
                return;
            }
            String frameUrl = frame.url();
            if (frameUrl != null && frameUrl.startsWith("chrome-error://")) {
                log.info("Detected chrome-error (link.dana.id) — ignoring (headless cannot open deep links)");
            }
        });
    }

    private static void navigateToOAuthUrl(Page page, String redirectUrl) {
        page.navigate(redirectUrl, new Page.NavigateOptions()
                .setTimeout(NAVIGATION_TIMEOUT_MS)
                .setWaitUntil(com.microsoft.playwright.options.WaitUntilState.DOMCONTENTLOADED));
    }

    private static boolean visibleNow(Locator locator) {
        return locator.isVisible(new Locator.IsVisibleOptions().setTimeout(0));
    }

    private static void fillPhoneNumber(Page page, String phoneNumber) {
        // Match Go phone selectors (normalized number with leading 0).
        String[] phoneSelectors = {
                "input.txt-input-phone-number-field",
                "input[type='tel']",
                "input[placeholder='12312345678']",
                "input[maxlength='13']",
                "input[maxlength='15']"
        };

        for (String selector : phoneSelectors) {
            Locator field = page.locator(selector).first();
            if (visibleNow(field)) {
                String current = field.inputValue();
                if (!isBlank(current)) {
                    log.info("Phone field already pre-filled by DANA: {} (skipping fill)", current);
                    return;
                }
                // Headless fill() often leaves LANJUTKAN disabled; type so the page gets input events.
                field.click();
                field.fill("");
                page.keyboard().type(phoneNumber, new Keyboard.TypeOptions().setDelay(50));
                log.info("Phone filled using selector: {} with value: {}", selector, phoneNumber);
                return;
            }
        }

        Locator label = page.locator("label.new-clearable-input.form-ipg-phonenumber").first();
        if (visibleNow(label)) {
            label.click();
            page.keyboard().type(phoneNumber, new Keyboard.TypeOptions().setDelay(50));
            log.info("Phone filled via label click + keyboard type");
            return;
        }

        log.warn("Could not determine phone field state");
    }

    private static void clickContinueAfterPhone(Page page) {
        String[] buttonTexts = {"LANJUTKAN", "Lanjutkan", "Next", "Continue"};
        for (String text : buttonTexts) {
            Locator button = page.getByRole(
                    com.microsoft.playwright.options.AriaRole.BUTTON,
                    new Page.GetByRoleOptions().setName(text).setExact(true)).first();
            if (visibleNow(button) && clickWhenEnabled(button, text)) {
                return;
            }
        }

        String[] buttonSelectors = {
                "button[type='submit']",
                "button.btn-primary",
                "button.next-button",
                ".btn-continue",
                ".btn-submit"
        };
        for (String selector : buttonSelectors) {
            Locator button = page.locator(selector).first();
            if (visibleNow(button) && clickWhenEnabled(button, selector)) {
                return;
            }
        }

        log.warn("Could not click LANJUTKAN button");
    }

    private static boolean clickWhenEnabled(Locator button, String label) {
        try {
            button.waitFor(new Locator.WaitForOptions()
                    .setState(com.microsoft.playwright.options.WaitForSelectorState.VISIBLE)
                    .setTimeout(LOCATOR_TIMEOUT_MS));
            // Wait until DANA enables the button after phone input events.
            pageWaitEnabled(button);
            button.click(new Locator.ClickOptions().setTimeout(LOCATOR_TIMEOUT_MS));
            log.info("Submit clicked via: {}", label);
            return true;
        } catch (RuntimeException e) {
            log.warn("Could not click {}: {}", label, e.getMessage());
            return false;
        }
    }

    private static void pageWaitEnabled(Locator button) {
        long deadline = System.currentTimeMillis() + LOCATOR_TIMEOUT_MS;
        while (System.currentTimeMillis() < deadline) {
            try {
                if (button.isEnabled()) {
                    return;
                }
            } catch (RuntimeException ignored) {
                // Keep polling.
            }
            sleepQuietly(200);
        }
        throw new RuntimeException("button still disabled after " + LOCATOR_TIMEOUT_MS + "ms");
    }

    private static void clickOptionalContinue(Page page) {
        Locator continueButton = page.locator("button.btn-continue.fs-unmask.btn.btn-primary").first();
        if (visibleNow(continueButton)) {
            continueButton.click();
            sleepQuietly(1000);
        }
    }

    private static void enterPin(Page page, String pin) {
        Locator pinField = page.locator(PIN_SELECTOR).first();
        boolean pinFilled = false;

        if (visibleNow(pinField)) {
            pinField.click();
            page.keyboard().type(pin);
            log.info("PIN entered via keyboard type");
            pinFilled = true;
        }

        if (!pinFilled) {
            try {
                pinField.waitFor(new Locator.WaitForOptions()
                        .setState(com.microsoft.playwright.options.WaitForSelectorState.VISIBLE)
                        .setTimeout(10000));
                pinField.click();
                page.keyboard().type(pin);
                log.info("PIN entered via WaitForSelector + keyboard type");
                pinFilled = true;
            } catch (RuntimeException ignored) {
                // Fall through.
            }
        }

        if (!pinFilled) {
            throw new RuntimeException("could not enter PIN: PIN field not visible (current URL: " + page.url() + ")");
        }
    }

    private static String waitForAuthCode(Page page, AtomicReference<String> capturedAuthCode) {
        // Already captured during PIN → redirect (Google often strips ?authCode= next).
        String fromListener = capturedAuthCode.get();
        if (!isBlank(fromListener)) {
            return fromListener;
        }

        String alreadyOnPage = extractAuthCodeFromRedirect(page.url());
        if (!isBlank(alreadyOnPage)) {
            return alreadyOnPage;
        }

        // Prefer waitForURL — matches the brief google.com/?authCode=... hop.
        try {
            page.waitForURL(AUTH_CODE_PATTERN,
                    new Page.WaitForURLOptions().setTimeout(AUTH_CODE_TIMEOUT_MS));
            String fromUrl = extractAuthCodeFromRedirect(page.url());
            if (!isBlank(fromUrl)) {
                return fromUrl;
            }
        } catch (RuntimeException e) {
            log.warn("waitForURL(authCode) timed out/failed: {}", e.getMessage());
        }

        fromListener = capturedAuthCode.get();
        if (!isBlank(fromListener)) {
            return fromListener;
        }

        String authCode = extractAuthCodeFromRedirect(page.url());
        if (!isBlank(authCode)) {
            return authCode;
        }

        throw new RuntimeException("could not capture authorization code (current URL: " + page.url() + ")");
    }

    private static String extractAuthCodeFromRedirect(String url) {
        if (isBlank(url)) {
            return null;
        }

        try {
            URI uri = URI.create(url);
            String fromQuery = queryParam(uri.getRawQuery(), "authCode");
            if (!isBlank(fromQuery)) {
                return decodeUrlComponent(fromQuery);
            }
        } catch (IllegalArgumentException ignored) {
            // Fall through.
        }

        return extractAuthCodeFromUrl(url);
    }

    static String extractAuthCodeFromUrl(String url) {
        if (isBlank(url)) {
            return null;
        }

        Matcher matcher = AUTH_CODE_PATTERN.matcher(url);
        if (matcher.find()) {
            return decodeUrlComponent(matcher.group(1));
        }

        if (url.contains("authCode=")) {
            String fragment = url.split("authCode=", 2)[1];
            return decodeUrlComponent(fragment.split("&")[0]);
        }

        return null;
    }

    private static String queryParam(String query, String name) {
        if (query == null || query.isEmpty()) {
            return null;
        }
        for (String pair : query.split("&")) {
            String[] parts = pair.split("=", 2);
            if (parts.length == 2 && name.equals(parts[0])) {
                return parts[1];
            }
        }
        return null;
    }

    private static String decodeUrlComponent(String value) {
        try {
            return URLDecoder.decode(value, StandardCharsets.UTF_8.name());
        } catch (UnsupportedEncodingException e) {
            throw new RuntimeException(e);
        }
    }

    private static boolean isBlank(String value) {
        return value == null || value.trim().isEmpty();
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
