# Endpoint inventory

The coverage map for this SDK. One row per endpoint: what it is, where it lives, how it is
encoded, and whether we have built it. Update the Status column **in the same PR as the code**
— a matrix that lags reality is worse than no matrix, because people act on it.

## Provenance

`developers.africastalking.com` sits behind a bot challenge and cannot be fetched
programmatically, so the rows below were derived from **Africa's Talking' own SDKs**, which are
first-party code written against the live API:

- [`africastalking-node.js`](https://github.com/AfricasTalkingLtd/africastalking-node.js) v0.8.4
  (last updated 2026-10-09) — the most actively maintained, and the primary source here.
- [`africastalking-go`](https://github.com/AfricasTalkingLtd/africastalking-go)
  (last updated 2026-03-15) — used for Payments, which the Node SDK no longer ships.

This is much stronger evidence than a third-party spec, but it is still not the official docs,
and the two SDKs disagree in places (noted inline). **No row has been confirmed against a live
sandbox call.** Verify before implementing, then set Verified to `yes` and record surprises in
Notes — the Notes column is the most valuable part of this table.

## Hosts

Africa's Talking uses a host per product, each with a sandbox twin at
`<service>.sandbox.africastalking.com`. `WithSandbox()` must switch all of them together.

| Key | Production | Sandbox | Products |
|---|---|---|---|
| `api` | `https://api.africastalking.com` | `https://api.sandbox.africastalking.com` | SMS, Airtime, Subscriptions, Application |
| `content` | `https://content.africastalking.com` | *falls back to `api`* | Premium SMS |
| `voice` | `https://voice.africastalking.com` | `https://voice.sandbox.africastalking.com` | Voice |
| `payments` | `https://payments.africastalking.com` | `https://payments.sandbox.africastalking.com` | Payments |
| `bundles` | `https://bundles.africastalking.com` | `https://bundles.sandbox.africastalking.com` | Mobile Data |
| `insights` | `https://insights.africastalking.com` | `https://insights.sandbox.africastalking.com` | Insights (post-v1) |
| `chat` | `https://chat.africastalking.com` | `https://chat.sandbox.africastalking.com` | Chat / WhatsApp (post-v1) |

**`/version1` is a path prefix, not part of the host.** It applies to most `api`-host endpoints
but *not* to `/checkout/token/create` or `/auth-token/generate`, which sit at the host root.
Modelling it as part of the host — as `hosts.go` currently does — breaks those two.

In sandbox, the Node SDK points `content` at the `api` host rather than
`content.sandbox.…`, so premium SMS cannot be tested against a dedicated sandbox host.

**Auth:** `apiKey` request header (the Node SDK spells it `apikey`; HTTP headers are
case-insensitive so either works), plus `username` as a body or query field.
Send `Accept: application/json` on every request regardless of request content type.

**Idempotency:** the Node SDK sends an optional `Idempotency-Key` header on airtime. This is
the only idempotency mechanism found anywhere in the SDKs — relevant to every retry decision.

**Success is `201`, not `200`.** Both SDKs treat `201 Created` as the success status for sends
and reject anything else. Our `resp.StatusCode < 200 || > 299` check covers it, but fixtures
should use 201.

## Status legend

| Value | Meaning |
|---|---|
| `done` | Implemented, fixtures captured, tests passing |
| `wip` | Someone is working on it — see the linked issue |
| `todo` | Unclaimed; good first issue material |
| `n/a` | Inbound callback, or not planned before v1 |

## SMS — host `api`

| Method | Path | Content type | Status | Verified | Notes |
|---|---|---|---|---|---|
| POST | `/version1/messaging` | `x-www-form-urlencoded` | `wip` | no | Bulk/standard send. Fields: `username`*, `to`* (**comma-joined string**, not an array), `message`*, `from`, `bulkSMSMode=1`, `enqueue` (`1`/`0`). Our `sms.go` currently posts **JSON** to `/messaging/bulk` — see Open questions #1. |
| GET | `/version1/messaging` | — (query) | `todo` | no | Fetch inbox. Query: `username`*, `lastReceivedId` (`0` from the start). Cursor-paged. Response: `SMSMessageData.Messages[]` with `linkId`, `text`, `to`, `from`, `id`, `date`. |

Per-recipient result codes are in [`sms_status.go`](../sms_status.go) as `SMSStatusCode`; a
successful response can still carry rejected recipients.

## Premium SMS and subscriptions

| Method | Path | Host | Content type | Status | Verified | Notes |
|---|---|---|---|---|---|---|
| POST | `/version1/messaging` | **`content`** | `x-www-form-urlencoded` | `todo` | no | Premium send — **a different host from bulk send**, same path. Fields: as bulk plus `bulkSMSMode=0`, `keyword`*, `linkId`, `retryDurationInHours`. |
| POST | `/checkout/token/create` | `api` | `x-www-form-urlencoded` | `todo` | no | **No `/version1` prefix.** Fields: `phoneNumber`*. Returns the `checkoutToken` for the next call. |
| POST | `/version1/subscription/create` | `api` | `x-www-form-urlencoded` | `todo` | no | Fields: `username`*, `shortCode`*, `keyword`*, `phoneNumber`*, `checkoutToken`*. |
| POST | `/version1/subscription/delete` | `api` | `x-www-form-urlencoded` | `todo` | no | Fields: `username`*, `shortCode`*, `keyword`*, `phoneNumber`*. No checkout token. |
| GET | `/version1/subscription` | `api` | — (query) | `todo` | no | Query: `username`*, `shortCode`*, `keyword`*, `lastReceivedId`. Cursor-paged. |
| POST | `/auth-token/generate` | `api` | — | `n/a` | no | **No `/version1` prefix.** Short-lived auth token; not needed while we authenticate by API key. |

`linkId` comes from an inbound subscription callback, so premium sending only works alongside
the webhook work in Stage 3.

## Airtime — host `api`

| Method | Path | Content type | Status | Verified | Notes |
|---|---|---|---|---|---|
| POST | `/version1/airtime/send` | `x-www-form-urlencoded` | `todo` | no | Fields: `username`*, `recipients`* (**JSON-encoded array in a single form field**; each object is `{phoneNumber, amount}` where amount is the display string `"KES 100"` — currency and value are **joined**, not separate), `maxNumRetry`. Exactly the case `form:"...,json"` exists for. Accepts an `Idempotency-Key` header. The official Go SDK uses `/version1/airtime` instead — see Open questions #2. |
| GET | `/query/transaction/find` | — (query) | `todo` | no | **No `/version1` prefix.** Airtime transaction lookup. |

## Voice — host `voice`

Content type is **not uniform** across this product.

| Method | Path | Content type | Status | Verified | Notes |
|---|---|---|---|---|---|
| POST | `/call` | `x-www-form-urlencoded` | `todo` | no | Fields: `username`*, `to`* (comma-joined), `from`*, `clientRequestId`. Asynchronous: success means **queued**; AT then fetches your callback for the call flow. |
| POST | `/queueStatus` | **`application/json`** | `todo` | no | Fields: `username`*, `phoneNumbers`. JSON, unlike `/call`. A POST despite being a read. |
| POST | `/mediaUpload` | **`application/json`** | `todo` | no | Fields: `username`*, `url`* (HTTP(S) URL of the file), `phoneNumber`*. A **URL reference**, not a multipart upload — no multipart support needed in the transport. |

Call-flow XML (`Say`, `Play`, `GetDigits`, `Dial`, `Record`, `Enqueue`, `Dequeue`,
`Conference`, `Redirect`, `Reject`) is served from **your** callback, so it has no row here.

## Payments — host `payments`

All JSON. Sourced from the official Go SDK; the Node SDK no longer ships Payments.

| Method | Path | Status | Verified | Notes |
|---|---|---|---|---|
| POST | `/mobile/checkout/request` | `todo` | no | C2B. Fields: `username`*, `productName`*, `phoneNumber`*, `currencyCode`*, `amount`*, `providerChannel`, `metadata`. **Asynchronous** — success means *accepted*; payment is confirmed only by a later webhook. |
| POST | `/mobile/b2c/request` | `todo` | no | Pays money out. Fields: `username`*, `productName`*, `recipients`* (array). **Never retry.** |
| POST | `/mobile/b2b/request` | `todo` | no | Fields: `username`*, `productName`*, `provider`*, `transferType`*, `currencyCode`*, `amount`*, `destinationChannel`*, `destinationAccount`*, `metadata`. |
| POST | `/bank/checkout/charge` | `todo` | no | Step 1 of 2; completed by `/bank/checkout/validate`. |
| POST | `/bank/checkout/validate` | `todo` | no | Step 2 — OTP validation. Ship both together or the payment silently never completes. |
| POST | `/bank/transfer` | `todo` | no | |
| POST | `/card/checkout/charge` | `todo` | no | Step 1 of 2. |
| POST | `/card/checkout/validate` | `todo` | no | Step 2 — OTP validation. |

`amount` is a JSON **number**. Do not model it as `float64` in Go — see the money rule in
[ROADMAP.md](../ROADMAP.md) Stage 6.

**Not found in either SDK**, so unlisted rather than unsupported: wallet balance/transfer/topup
stash, and the payment query endpoints (find transaction, fetch product transactions, fetch
wallet transactions). Research before Stage 6.

## Application — host `api`

| Method | Path | Content type | Status | Verified | Notes |
|---|---|---|---|---|---|
| GET | `/version1/user` | — (query) | `todo` | no | Account data including balance. Query: `username`*. Confirmed in both SDKs. Balance arrives as the display string `"KES 1234.50"` — parse to a typed amount but keep the raw string. |

## Mobile Data — host `bundles`

| Method | Path | Content type | Status | Verified | Notes |
|---|---|---|---|---|---|
| POST | `/mobile/data/request` | `application/json` | `n/a` | no | Fields: `username`*, `productName`*, `recipients`* (array). Post-v1. |
| GET | `/query/transaction/find` | — (query) | `n/a` | no | Query: `username`, `transactionId`. |
| GET | `/query/wallet/balance` | — (query) | `n/a` | no | Query: `username`. |

## Inbound callbacks (no outbound endpoint)

HTTP requests **Africa's Talking makes to you**. These belong to `webhook/` and `ussd/`, not to
a service method. Each needs a captured payload in `testdata/` before its parser is written.

| Callback | Status | Notes |
|---|---|---|
| USSD session | `todo` | Form-encoded POST: `sessionId`, `serviceCode`, `phoneNumber`, `text`. `text` replays the **whole accumulated input** (`1*2*3`). Reply with a body prefixed `CON ` or `END `. |
| SMS delivery report | `todo` | The only authoritative delivery signal — a successful send means accepted, not delivered. |
| Incoming SMS | `todo` | Messages to your short code. |
| Premium subscription | `todo` | Source of the `linkId` needed for premium sends. |
| Voice call flow | `todo` | AT fetches your URL mid-call and expects call-action XML back. |
| Voice events | `todo` | Receives the `clientRequestId` passed to `POST /call`. |
| Payment notification | `todo` | The only confirmation an async checkout succeeded. Handlers must be **idempotent**: notifications can arrive more than once. |

## Open questions

Resolve each against the official docs plus one sandbox call, then fold the answer in and
delete the entry.

1. **`/messaging` vs `/messaging/bulk`.** Both official SDKs post form-encoded to
   `/version1/messaging` with `to` comma-joined and `bulkSMSMode=1`. Neither calls
   `/messaging/bulk`. The Node SDK carries this comment at the send path:

   > `TO DO -> once the sandbox for new bulk SMS is live: Remove "from" entirely, Change the
   > endpoint to the new bulk SMS endpoint`

   So a JSON bulk endpoint appears to be **planned but not yet live**, and our `sms.go` targets
   it. Confirm, then either move to `/version1/messaging` form-encoded, or keep `/messaging/bulk`
   and document that it needs an account with the new bulk SMS enabled. **Highest priority in
   this file** — it is the one endpoint we have implemented.
2. **Airtime path.** Node uses `/version1/airtime/send`; the official Go SDK uses
   `/version1/airtime`. Node is the more recently maintained, so prefer it, but verify.
3. **`maskedNumber` and `telco`** appear in our `SendSMSRequest` but in neither official SDK nor
   any documented schema. Confirm they exist on some send variant, or remove them.
4. **`enqueue` semantics.** Node sends `1` for true, `0` for explicit false, and omits the field
   when unset — three distinct states. Confirm whether omitted and `0` differ at the API; if
   they do, the public type needs to be `*bool`, not `bool`.
5. **`content` host in sandbox.** Node falls back to the `api` host rather than
   `content.sandbox.…`. Confirm whether a premium sandbox host exists.
6. **Error body shape.** `extractMessage` probes `errMessage`, `message`, `description`,
   `status` and `SMSMessageData.Message`. Capture real failures (bad API key, bad sender ID,
   invalid number, empty balance) and keep only the fields that appear.
7. **Idempotency beyond airtime.** Airtime accepts `Idempotency-Key`. Determine whether any
   payments endpoint does — it decides whether a B2C payout can ever be safely retried.
8. **Rate limits.** No published numeric limit found; throughput appears carrier- and
   account-governed. Confirm whether any endpoint returns `429`, which decides whether the
   transport's `Retry-After` handling is ever exercised.
