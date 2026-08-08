const POPUP_FEATURES = "width=480,height=760,menubar=no,toolbar=no,location=yes,status=no";

// PayPetal has no embeddable checkout (their payment page sends
// X-Frame-Options: DENY, and there's no JS SDK) — a popup is the closest
// thing to "stay on the site" that's actually achievable: the main
// GigPurse tab never navigates away, only a separate window does.
//
// `/contracts/pending` (where PayPetal redirects back to) detects it's
// running inside this popup and posts a message + closes itself once the
// payment is confirmed — this is what fires `onClose`. If the user closes
// the popup manually instead, the fallback poll on `popup.closed` catches
// that too, so `onClose` always eventually fires either way.
export function openPaymentPopup(url, { onClose } = {}) {
  const popup = window.open(url, "paypetal-checkout", POPUP_FEATURES);
  if (!popup) {
    // Blocked by a popup blocker (some browsers are strict about `window.open`
    // calls that happen after an awaited API response, not a direct click) —
    // fall back to the old same-tab redirect rather than leaving the user stuck.
    window.location.href = url;
    return;
  }
  popup.focus();

  let done = false;
  function finish() {
    if (done) return;
    done = true;
    clearInterval(poll);
    window.removeEventListener("message", handleMessage);
    onClose?.();
  }

  function handleMessage(event) {
    if (event.origin !== window.location.origin) return;
    if (event.data?.source !== "gigpurse-payment") return;
    finish();
  }
  window.addEventListener("message", handleMessage);

  const poll = setInterval(() => {
    if (popup.closed) finish();
  }, 700);
}
