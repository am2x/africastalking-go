# Contributing

Thanks for considering a contribution. This is an unofficial, community-maintained Go SDK for
the [Africa's Talking](https://africastalking.com) API, and the fastest way to help is to pick
an unimplemented endpoint from [`docs/endpoints.md`](docs/endpoints.md) and build it.

Most of this document is one worked example: **how to add an endpoint**. The SMS service is the
reference implementation. If you copy its shape and satisfy the checklist at the end, your PR
will be straightforward to review.

## Getting set up

You need Go 1.23 or newer (the floor is the `go` directive in `go.mod`). There are no other
build dependencies — the library itself has **zero third-party dependencies** and we intend to
keep it that way.

```bash
git clone https://github.com/am2x/africastalking-go.git
cd africastalking-go
make check
```

`make check` is the one command to remember. It runs exactly what CI runs — formatting, vet,
lint and the tests under the race detector — so if it is green locally, CI will be green too.
`make help` lists the other targets.

The unit tests need **no credentials and no network access**. Every test runs against a local
HTTP server. If you ever find yourself needing a real API key to run `make test`, something has
gone wrong.

`golangci-lint` is needed for `make lint`. Install the version pinned in the Makefile:

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4
```

A note on that pin: golangci-lint type-checks through `x/tools`, which cannot read export data
from a Go release newer than its own. CI therefore pins the lint job's Go to the floor in
`go.mod`. **If you raise the Go floor, bump `GOLANGCI_VERSION` in the same commit**, or linting
breaks with errors that look like they come from your code.

## How the code is laid out

```
africastalking.go      the client, New(), service wiring
options.go             WithCredentials, WithSandbox, WithBaseURL
hosts.go               the per-product hosts and their sandbox twins
errors.go              APIError, ValidationErr, the sentinel errors
phone.go               phone number normalization
sms.go                 the SMS service: one file per product
sms_types.go           its public request and response types
sms_status.go          its per-recipient status codes
internal/transport/    HTTP, retries, backoff - the only code that knows HTTP exists
internal/encoding/     struct to form or JSON
internal/testutil/     the test HTTP server and fixture helpers
testdata/              recorded API responses, one directory per product
docs/endpoints.md      the coverage map
```

Two rules about this layout:

**Services live in the root package, one file per product.** There is no `pkg/` directory: in a
library the repo root *is* the import path. A new product is `voice.go`, `voice_types.go`,
`voice_test.go` — not a subpackage. See
[`docs/adr/0001-single-root-package.md`](docs/adr/0001-single-root-package.md).

**Nothing under `internal/` may import the root package.** The dependency runs one way only. If
something in `internal/` seems to need a root type, make it generic (`any` or a narrow
interface), as `encoding.EncodeJSON` does. This is why error mapping lives in the root package
and not in the transport.

## Adding an endpoint: a worked example

Say you are adding `POST /version1/airtime/send`.

### 1. Start from the inventory

Find the row in [`docs/endpoints.md`](docs/endpoints.md). It tells you the host, path, method,
content type and field names. **Verify it against Africa's Talking' own docs before you start** —
every row is marked `Verified: no` until someone has confirmed it with a real sandbox call. When
you confirm one, flip the column and record anything surprising in the Notes.

If the row disagrees with reality, fixing the row is as valuable as the code.

### 2. Public types in `airtime_types.go`

Exported, documented, and named for Go rather than for the wire:

```go
// SendAirtimeRequest is the input to [AirtimeService.Send].
type SendAirtimeRequest struct {
    // Recipients are the numbers to credit, each with its own amount.
    Recipients []AirtimeRecipient
}
```

Do **not** put wire JSON tags on public types. They describe your API, not Africa's Talking'.
The wire shape belongs to an unexported payload struct (step 4), which is what lets us inject
`username` from the client and keep field names stable if the API changes.

### 3. The service, in `airtime.go`

```go
// AirtimeService provides access to the Africa's Talking airtime endpoints.
// Access it through the Airtime field of [AfricasTalking].
type AirtimeService struct{ at *AfricasTalking }

const airtimePath = "/version1/airtime/send"
```

Wire it up in `New()` in `africastalking.go`, after the options are applied:

```go
at.Airtime = &AirtimeService{at: at}
```

and add the exported field to the `AfricasTalking` struct. Users then reach it as
`at.Airtime.Send(...)`.

### 4. The method

Every service method has the same five parts, in this order:

```go
func (s *AirtimeService) Send(ctx context.Context, req SendAirtimeRequest) (SendAirtimeResponse, error) {
    // 1. Validate locally. Cheap checks turn a confusing remote 4xx into an
    //    immediate, precise error. Return *ValidationErr, never *APIError.
    if len(req.Recipients) == 0 {
        return SendAirtimeResponse{}, &ValidationErr{
            Field:   "Recipients",
            Message: "at least one recipient is required",
        }
    }

    // 2. Build the unexported wire payload. This is where username comes from
    //    the client and phone numbers are normalized.
    payload := airtimePayload{Username: s.at.userName /* ... */}

    // 3. Encode for this endpoint's content type.
    body, err := encoding.EncodeForm(payload)
    if err != nil {
        return SendAirtimeResponse{}, err
    }

    // 4. Hand it to the transport. Note Retryable.
    resp, err := s.at.client.Do(ctx, transport.Request{
        Method:      http.MethodPost,
        URL:         s.at.apiURL(airtimePath),
        ContentType: transport.ContentTypeFormEncoded,
        UserAgent:   transport.UserAgent,
        Body:        body,
        Retryable:   false,
    }, s.at.apiKey)
    if err != nil {
        return SendAirtimeResponse{}, fmt.Errorf("africastalking: send airtime: %w", err)
    }

    // 5. Map the status, then decode.
    if resp.StatusCode < 200 || resp.StatusCode > 299 {
        return SendAirtimeResponse{}, newAPIError(airtimePath, resp)
    }
    var out SendAirtimeResponse
    if err := json.Unmarshal(resp.Body, &out); err != nil {
        return SendAirtimeResponse{}, fmt.Errorf("africastalking: decode airtime response: %w", err)
    }
    return out, nil
}
```

Four things that are easy to get wrong:

- **`ctx` is always the first parameter**, and it must reach the transport. A method that accepts
  a context and ignores it is worse than one that takes none, because the signature promises
  cancellation that does not work.
- **`Retryable` is a money decision, not a performance one.** Set it `false` for anything that
  creates, sends, or moves money: Africa's Talking has no idempotency key on most endpoints, so
  a resend after a timeout can double-send and double-charge. Reads (`Fetch`, balance lookups)
  set it `true`.
- **Pick the right host accessor.** `apiURL`, `contentURL`, `voiceURL`, `paymentsURL` — they are
  different hosts, and the inventory says which. Using the wrong one produces a 404 that reads
  like an auth problem.
- **Non-2xx becomes `*APIError` in the root package**, never in the transport. Some Africa's
  Talking endpoints describe a failure in the body of an otherwise successful response, so check
  the payload too where the inventory says so.

### 5. Fixtures in `testdata/airtime/`

Capture real responses from sandbox, then scrub: no real phone numbers (use `+254700000000`),
no credentials. Save one per documented outcome, **including the failures** — a partial failure
and a non-JSON error body are where the bugs live, and the happy path never catches them.

### 6. Tests in `airtime_test.go`

Use the helpers. `newTestClient` in `helpers_test.go` gives you a client pointed at a local
server:

```go
at, srv := newTestClient(t, testutil.Fixture(t, http.StatusOK, "airtime/send_success.json"))
```

`internal/testutil` provides `Respond` (a fixed status and body), `Fixture` (serve a file from
`testdata/`) and `Sequence` (a different response per request, for retry and paging tests). The
server records every request, so assert on **what was sent**, not only on what came back:

```go
got := srv.LastRequest(t)
if got.Path != airtimePath { /* ... */ }

var sent map[string]any
got.DecodeJSON(t, &sent)
if sent["username"] != "testuser" { /* ... */ }
```

That matters because most SDK bugs are a wrong field name, host, or content type — and a test
that only checks the decoded response cannot see any of them. Real examples from this repo:
asserting the outgoing `keyword` caught it being populated from the wrong field, and asserting
the query string caught `Fetch` silently sending no parameters.

Table-driven tests, one row per fixture. Three tests are worth writing every time:

- **validation short-circuits** — assert `srv.Count() == 0`, which is the only way to prove no
  request was made rather than that the error happened to be right
- **the retry policy** — `Sequence(Respond(503, ...), success)` then assert `srv.Count()`: `1`
  for a send, `2` for a read
- **a cancelled context** — `errors.Is(err, context.Canceled)`

### 7. Documentation

Doc comments on every exported symbol, starting with the symbol's name (`// Send submits …`).
Document the surprises, not the signature: if a nil error does not mean what a reader would
assume, say so. Add a runnable `Example` to `example_test.go` if the usage is not obvious —
those are compiled by `go test`, so unlike README snippets they cannot go stale.

Then update `docs/endpoints.md` and add a `CHANGELOG.md` entry under `Unreleased`.

## Conventions

- **Branches:** `main` is always green; `dev` is the integration branch. Work on
  `feat/airtime-send` or `fix/retry-jitter`.
- **Commits:** [Conventional Commits](https://www.conventionalcommits.org) —
  `feat(airtime): add send endpoint`, `fix(transport): attach query before building the request`.
- **Naming:** Go initialism rules. `SenderID` not `SenderId`, `APIError` not `ApiError`,
  `HTTPClient` not `HttpClient`. The linter enforces this.
- **Money is never a `float64`.** Binary floating point cannot represent 0.10 exactly, and in a
  payments SDK that drift is someone's money. Use a decimal string or minor units. The form
  encoder deliberately rejects float fields.
- **No logging from the library.** Return errors; let the caller decide whether and where to
  log.
- **Never put the API key in an error, a log line, or a struct that might be printed.** Errors
  get logged, which makes them the most likely place to leak a credential.
- **The client is immutable after `New()` returns**, which is what makes it safe for concurrent
  use. Keep it that way: no setters, no mutable package state.

## Opening a pull request

Run `make check` first. Then make sure every box below is ticked — the checklist *is* the review,
so a PR that satisfies it needs very little discussion.

- [ ] Request and response types in `<product>_types.go`, exported and documented
- [ ] Method on the service struct, `ctx context.Context` first, returning `(T, error)`
- [ ] Local validation of required fields, returning `*ValidationErr`
- [ ] Correct host accessor **and** correct content type for that endpoint
- [ ] `Retryable` set deliberately, with a comment if it is `true`
- [ ] Non-2xx mapped to `*APIError` via `newAPIError`
- [ ] At least one fixture in `testdata/<product>/` per documented outcome, failures included
- [ ] Table-driven tests asserting the outgoing request, not just the response
- [ ] A test proving validation never reaches the network (`srv.Count() == 0`)
- [ ] Doc comments on every exported symbol; an `Example` where the usage is not obvious
- [ ] `docs/endpoints.md` status updated in this PR
- [ ] `CHANGELOG.md` entry under `Unreleased`
- [ ] `make check` passes

## Claiming work

Comment on the issue and it will be assigned to you. If you go quiet for a couple of weeks it
may be unassigned so someone else can pick it up — no hard feelings, just say the word if you
want it back.

Expect a first response within a few days. If a week goes by with silence, bump the thread;
that is a lapse on our side, not an imposition on yours.

Small PRs are easier to review than large ones. One endpoint per PR is ideal. If you want to
discuss an approach before writing code, open an issue first — that is always cheaper than
rewriting a finished PR.

## Reporting a security issue

Please do not open a public issue for a vulnerability. See [SECURITY.md](SECURITY.md).
