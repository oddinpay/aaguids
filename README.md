# aaguids

Detect the passkey provider's name, icon, and ID. This project supports both hardware and software vendors. The `aaguids.json` file is production-ready and up to date, with more providers to be added eventually. This includes officially mentioned `authenticator` providers and third-party providers.

<br>

## Prerequisites

Before running aaguids, ensure your system meets the following requirements:

- **[Go](https://go.dev/dl)** : version 1.23+ is required

<br>

## ⚡ CDN

- https://cdn.oddinpay.com/aaguids.json

<br>

## Usage (Demo)

```sh
git clone https://github.com/oddinpay/aaguids

cd aaguids

go mod tidy

go run .

```

### Output

```bash

[FOUND] 54d9fee8-e621-4291-8b18-7157b99c5bec -> HID Crescendo Enabled data:image/svg+xml;base64
[FOUND] 560a780cb6ae4f03b110082f856425b4 -> KQC QuKey Bio FIDO2 Authenticator data:image/svg+xml;base64

```

<br>

## References

- [FIDO Alliance](https://fidoalliance.org)
- [Passkeys.dev](https://passkeys.dev)
- [Web Authentication API](https://developer.mozilla.org/en-US/docs/Web/API/Web_Authentication_API)
- [W3](https://www.w3.org/TR/webauthn-3)
