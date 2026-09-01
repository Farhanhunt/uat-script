import asyncio
import json
import os
import re
import time
from urllib.parse import parse_qs, unquote, urlparse
from uuid import uuid4

from dana.widget.v1.models import Oauth2UrlData, Oauth2UrlDataSeamlessData
from dana.widget.v1.util import Util

DEFAULT_PHONE = "083811223355"
DEFAULT_PIN = "181818"

IPHONE_13 = {
    "user_agent": (
        "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) "
        "AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Mobile/15E148 Safari/604.1"
    ),
    "viewport": {"width": 390, "height": 844},
    "device_scale_factor": 3,
    "is_mobile": True,
    "has_touch": True,
}


def _require_async_playwright():
    """Lazy import so mandatory-only collection works without Playwright installed."""
    try:
        from playwright.async_api import async_playwright

        return async_playwright
    except ImportError as e:
        raise ImportError(
            "playwright is required for OAuth UI automation. "
            "Install test/python/requirements.txt and run playwright install chromium."
        ) from e


def normalize_mobile_number(mobile: str) -> str:
    digits = re.sub(r"\D+", "", mobile or "")
    if not digits:
        return DEFAULT_PHONE
    if digits.startswith("62"):
        return "0" + digits[2:]
    return digits


def phone_digits_for_input(phone_number: str) -> str:
    """Local digits for +62 IPG inputs (no leading 0)."""
    phone = normalize_mobile_number(phone_number)
    return phone[1:] if phone.startswith("0") else phone


async def _wait_until_enabled(locator, timeout_ms: int = 15000) -> bool:
    deadline = time.monotonic() + (timeout_ms / 1000)
    while time.monotonic() < deadline:
        try:
            if await locator.is_enabled():
                return True
        except Exception:
            pass
        await asyncio.sleep(0.2)
    return False


async def _type_phone_into_field(page, locator, phone_number: str, selector: str) -> None:
    """Click + keyboard type so React enables CONTINUE (fill() often does not)."""
    digits = phone_digits_for_input(phone_number)
    await locator.click()
    try:
        await locator.fill("")
    except Exception:
        pass
    await page.keyboard.type(digits, delay=50)
    print(f"Phone typed using selector: {selector} with value: {digits}")


async def _fill_phone_number(page, phone_number: str) -> bool:
    phone_selectors = [
        "input.txt-input-phone-number-field",
        ".desktop-input>.txt-input-phone-number-field",
        "input[type='tel']",
        "input[placeholder*='12345678' i]",
        "input[placeholder*='phone' i]",
        "input[class*='phone' i]",
        "label.new-clearable-input.form-ipg-phonenumber",
    ]

    try:
        await page.wait_for_selector(
            "input.txt-input-phone-number-field, input[type='tel'], label.form-ipg-phonenumber",
            state="visible",
            timeout=15000,
        )
    except Exception:
        pass

    for sel in phone_selectors:
        loc = page.locator(sel).first
        try:
            if not await loc.is_visible(timeout=2000):
                continue
        except Exception:
            continue

        if sel.endswith("form-ipg-phonenumber"):
            await loc.click()
            await page.keyboard.type(phone_digits_for_input(phone_number), delay=50)
            print("Phone typed via label.new-clearable-input.form-ipg-phonenumber")
            return True

        current_val = (await loc.input_value()).strip()
        if current_val:
            print(f"Phone field already pre-filled by DANA: {current_val} (skipping fill)")
            return True

        await _type_phone_into_field(page, loc, phone_number, sel)
        return True

    textbox = page.get_by_role("textbox").first
    try:
        if await textbox.is_visible(timeout=2000):
            await _type_phone_into_field(page, textbox, phone_number, "role=textbox")
            return True
    except Exception:
        pass

    return False


async def _click_continue_button(page) -> bool:
    print("Looking for CONTINUE / LANJUTKAN button...")
    button_patterns = [
        re.compile(r"^continue$", re.I),
        re.compile(r"^lanjutkan$", re.I),
        re.compile(r"^next$", re.I),
    ]
    for pattern in button_patterns:
        loc = page.get_by_role("button", name=pattern).first
        try:
            if not await loc.is_visible(timeout=2000):
                continue
            if not await _wait_until_enabled(loc):
                print(f"Button matched {pattern.pattern} but stayed disabled")
                continue
            await loc.click(timeout=10000)
            print(f"Submit clicked via role+name: {pattern.pattern}")
            return True
        except Exception as e:
            print(f"Could not click button {pattern.pattern}: {e}")

    css_selectors = [
        "button[type='submit']",
        "button.btn-primary",
        "button.next-button",
        ".btn-continue",
        ".btn-submit",
        "button:has-text('CONTINUE')",
        "button:has-text('Continue')",
        "button:has-text('LANJUTKAN')",
    ]
    for sel in css_selectors:
        loc = page.locator(sel).first
        try:
            if not await loc.is_visible(timeout=2000):
                continue
            if not await _wait_until_enabled(loc):
                continue
            await loc.click(timeout=10000)
            print(f"Submit clicked via selector: {sel}")
            return True
        except Exception:
            pass

    print("Warning: could not click CONTINUE button")
    return False


def extract_auth_code_from_url(url: str):
    try:
        parsed = urlparse(url)
        params = parse_qs(parsed.query)
        return params.get("auth_code", [None])[0] or params.get("authCode", [None])[0]
    except Exception:
        return None


def extract_mobile_from_url(url: str) -> str:
    try:
        parsed = urlparse(url)
        seamless_data = parse_qs(parsed.query).get("seamlessData", [None])[0]
        if seamless_data:
            json_data = json.loads(unquote(seamless_data))
            return normalize_mobile_number(json_data.get("mobile", DEFAULT_PHONE))
    except Exception as e:
        print(f"Error extracting mobile number: {e}")
    return DEFAULT_PHONE


def get_redirect_oauth_url(phone_number: str | None = None) -> str:
    """Generate OAuth URL — aligned with Go GetRedirectOauthUrl / widget.GenerateOauthUrl."""
    phone = normalize_mobile_number(phone_number or DEFAULT_PHONE)
    redirect_url = os.environ.get("REDIRECT_URL_OAUTH") or "https://google.com"

    # Only include mobileNumber to pre-fill the phone field.
    # Extra seamless fields can push DANA into /app/ login instead of merchant redirect.
    seamless_data = Oauth2UrlDataSeamlessData(mobile_number=phone)
    oauth_data = Oauth2UrlData(
        external_id=str(uuid4()),
        merchant_id=os.environ.get("MERCHANT_ID", ""),
        redirect_url=redirect_url,
        seamless_data=seamless_data,
        scopes=[Util.generate_scopes()],
    )

    oauth_url = Util.generate_oauth_url(oauth_data)
    print(f"RedirectOauthUrl: {oauth_url}")
    return oauth_url


async def _automate_oauth_flow(page, phone_number: str, pin: str, oauth_url: str):
    """Single OAuth attempt — aligned with Go automateOAuthFlow / Node automate-oauth.js."""

    async def on_framenavigated(frame):
        if frame.url.startswith("chrome-error://"):
            print("Detected chrome-error — going back to restore DANA page")
            await asyncio.sleep(0.3)
            try:
                await page.go_back(wait_until="domcontentloaded", timeout=8000)
                print("Successfully restored page after chrome-error")
            except Exception as e:
                print(f"GoBack from chrome-error failed: {e}")

    page.on("framenavigated", lambda frame: asyncio.create_task(on_framenavigated(frame)))

    await page.goto(oauth_url, wait_until="domcontentloaded", timeout=60000)
    await page.wait_for_timeout(2000)

    pin_selector = (
        ".txt-input-pin-field, input[maxlength='6'][inputmode='numeric'], input[type='password']"
    )
    is_pin_visible = await page.locator(pin_selector).first.is_visible(timeout=5000)

    if not is_pin_visible:
        # seamlessData pre-fills phone asynchronously on legacy pages.
        if "seamlessData=" in oauth_url:
            await page.wait_for_timeout(2000)

        phone_filled = await _fill_phone_number(page, phone_number)
        if not phone_filled:
            print("Warning: could not determine phone field state")

        await page.wait_for_timeout(1000)
        await _click_continue_button(page)

        await page.wait_for_timeout(3000)

        continue_loc = page.locator("button.btn-continue.fs-unmask.btn.btn-primary").first
        try:
            if await continue_loc.is_visible(timeout=2000) and await _wait_until_enabled(continue_loc):
                await continue_loc.click(timeout=10000)
                await page.wait_for_timeout(1000)
        except Exception:
            pass

        try:
            await page.wait_for_selector(pin_selector, state="attached", timeout=15000)
        except Exception:
            print("Timeout waiting for PIN field after phone submit")

    pin_filled = False
    pin_loc = page.locator(pin_selector).first
    if await pin_loc.is_visible():
        await pin_loc.click()
        await page.keyboard.type(pin)
        print("PIN entered via keyboard type")
        pin_filled = True

    if not pin_filled:
        try:
            el = await page.wait_for_selector(pin_selector, state="visible", timeout=10000)
            await el.click()
            await page.keyboard.type(pin)
            print("PIN entered via waitForSelector + keyboard type")
            pin_filled = True
        except Exception:
            pass

    if not pin_filled:
        raise RuntimeError("could not enter PIN: PIN field not visible")

    auth_code = None
    deadline = time.time() + 30

    def capture_auth_code(url: str):
        nonlocal auth_code
        parsed = urlparse(url)
        if parsed.netloc.endswith("google.com") or "authCode" in parse_qs(parsed.query):
            code = parse_qs(parsed.query).get("authCode", [None])[0]
            if code:
                auth_code = code

    page.on("framenavigated", lambda frame: capture_auth_code(frame.url))

    while time.time() < deadline and not auth_code:
        capture_auth_code(page.url)
        if auth_code:
            break
        await asyncio.sleep(0.5)

    if not auth_code:
        auth_code = extract_auth_code_from_url(page.url)

    if not auth_code:
        raise RuntimeError("could not capture authorization code")

    print(f"OAuth flow completed successfully, auth code: {auth_code}")
    return auth_code


async def automate_oauth_simple(
    phone_number=None,
    pin=None,
    oauth_url=None,
    max_retries=3,
    ci_mode=False,
):
    async_playwright = _require_async_playwright()
    mobile_number = normalize_mobile_number(phone_number or DEFAULT_PHONE)
    used_pin = pin or DEFAULT_PIN
    fixed_oauth_url = oauth_url

    if ci_mode:
        max_retries = 2

    launch_args = [
        "--disable-web-security",
        "--disable-features=IsolateOrigins",
        "--disable-site-isolation-trials",
        "--disable-blink-features=AutomationControlled",
        "--no-sandbox",
        "--disable-dev-shm-usage",
    ]
    if ci_mode:
        launch_args.extend(
            [
                "--disable-setuid-sandbox",
                "--disable-background-timer-throttling",
                "--disable-renderer-backgrounding",
            ]
        )

    for attempt in range(max_retries):
        print(f"\nOAuth Attempt {attempt + 1}/{max_retries}")
        print(f"Using mobile: {mobile_number}")
        print(f"Using PIN: {used_pin}")

        redirect_url = fixed_oauth_url or get_redirect_oauth_url(mobile_number)

        async with async_playwright() as p:
            browser = await p.chromium.launch(headless=True, args=launch_args)
            context = await browser.new_context(
                user_agent=IPHONE_13["user_agent"],
                viewport=IPHONE_13["viewport"],
                device_scale_factor=IPHONE_13["device_scale_factor"],
                is_mobile=IPHONE_13["is_mobile"],
                has_touch=IPHONE_13["has_touch"],
                locale="id-ID",
                geolocation={"longitude": 106.8456, "latitude": -6.2088},
                permissions=["geolocation"],
            )
            context.set_default_timeout(30000)
            page = await context.new_page()

            try:
                auth_code = await _automate_oauth_flow(
                    page, mobile_number, used_pin, redirect_url
                )
                await browser.close()
                return auth_code
            except Exception as e:
                print(f"Attempt {attempt + 1} failed with error: {e}")
            finally:
                await browser.close()

        if attempt < max_retries - 1:
            await asyncio.sleep(2 if ci_mode else 3)

    print(f"All {max_retries} attempts failed!")
    return None


async def automate_oauth(phone_number=None, pin=None, show_log=True):
    ci_mode = os.getenv("CI") is not None or os.getenv("GITLAB_CI") is not None
    return await automate_oauth_simple(phone_number, pin, max_retries=3, ci_mode=ci_mode)


if __name__ == "__main__":
    code = asyncio.run(automate_oauth_simple())
    print(f"Final result - Auth code: {code}")
