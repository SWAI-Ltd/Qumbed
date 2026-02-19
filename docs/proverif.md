# ProVerif formal verification

The Qumbed application-layer protocol (Subscribe → Publish → Relay → Message) is modeled and verified with [ProVerif](https://bblanche.gitlabpages.inria.fr/proverif/) in the **symbolic (Dolev–Yao) model**. Verification **passes** for both payload secrecy and correspondence under the assumptions below.

## Verification results

When you run `proverif proverif/qumbed.pv`:

| Query | Result | Meaning |
|--------|--------|--------|
| **not attacker(payload[])** | **true** | The payload is never derivable by the attacker (secrecy). |
| **event(subscriberReceived(x)) ==> event(publisherSent(x))** | **true** | Every value the subscriber receives was sent by the publisher (receive ⇒ send). |

Both queries are **true**; the protocol model is verified.

## Alignment with the main project

The model is based on the real Qumbed design and does not add protocol features that are not in the project:

- **Crypto:** Matches [internal/crypto/box.go](https://github.com/SWAI-Ltd/Qumbed/blob/main/internal/crypto/box.go) and [wire-protocol.md §6](wire-protocol.md). NaCl box is modeled as **authenticated encryption** `boxenc(m, recipient_pk, sender_sk)` / `boxdec(ciphertext, sender_pk, recipient_sk)`: only the recipient decrypts, and only the sender could have created the ciphertext. There is **no separate signature** in the model or in the real protocol; authenticity comes from the box.
- **Wire format:** Matches the spec. Subscribe: `(topic, public_key)`. Publish/Message: `(topic, encrypted_payload, sender_public_key)`. The relay forwards these shapes; it does not read payloads.
- **Publisher:** Encrypts only when the received key is the honest subscriber’s key (model of “encrypt only for a key you trust,” as in out-of-band key distribution).
- **Subscriber:** Accepts only when the message is from the honest publisher’s key (`pkSender = pkPubHonest`). This is a **deployment assumption** (trust one publisher key), not an extra protocol mechanism; in the real protocol the subscriber could enforce the same policy by ignoring messages from unknown sender keys.

## Model assumptions (no extra protocol)

1. **One honest subscriber key** — One key pair for the subscriber; the publisher encrypts only for that key. Matches the intended use: the publisher knows the subscriber’s key out-of-band.
2. **One honest publisher key** — One key pair for the publisher; the subscriber only accepts messages when `sender_public_key` is that key. Matches a deployment where the subscriber trusts a single publisher (e.g. one device or service).
3. **Authenticated encryption** — The box primitive binds the ciphertext to the sender; no signature field. Same as the real protocol.
4. **Relay** — Forwards frames without decrypting; zero-knowledge as in the project.

Nothing in the model (signatures, extra messages, or extra crypto) goes beyond what the main project specifies; the only extra is the **honest single subscriber and single publisher** for the proof.

## What is verified

- **Payload secrecy:** The attacker (and the relay) never learn the payload. Only the subscriber’s secret key can decrypt.
- **Correspondence:** If the subscriber successfully decrypts a value `x`, then the honest publisher sent `x`. So “received” implies “sent” for that payload.

## What is not modeled

- TLS/QUIC (transport is a public channel; the model is conservative).
- mDNS discovery, schema validation, rate limiting, implementation bugs.

## How to run

1. **Install ProVerif:** e.g. `opam install proverif`, or [download 2.05](https://bblanche.gitlabpages.inria.fr/proverif/), or use the [online demo](http://proverif24.paris.inria.fr/).
2. **Run:** From the project root, `proverif proverif/qumbed.pv` or `./scripts/verify_protocol.sh`.
3. **Expect:** Both queries reported as **true** in the verification summary.

## Files

| File | Purpose |
|------|--------|
| `proverif/qumbed.pv` | ProVerif model: crypto (boxenc/boxdec), processes, queries. |
| `scripts/verify_protocol.sh` | Runs ProVerif and exits 0 only if both queries pass. |

## Badge

When verification passes, use the “ProVerif verified” badge in the README (see README section “Formal verification”).
