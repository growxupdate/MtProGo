# Changelog

## v8.0.0-dev

- Added pure MTProto user account phone-code login foundation.
- Added `auth.sendCode` and `auth.signIn` helpers.
- Added `examples/mtproto_account_login`.
- Added `updates.getState` helper to verify user/bot authorization state.
- Kept the repository dependency-free and free of external Telegram client libraries.

## v8.0.0-dev

- Added pure MTProto encrypted client session helper.
- Added `auth.importBotAuthorization` over encrypted MTProto.
- Added raw `messages.sendMessage` builder/helper for known `InputPeer` values.
- Added `examples/mtproto_bot_login`.
- Added `examples/mtproto_send_message`.
- Updated API layer to 214.
- Kept the repository dependency-free and free of external Telegram client libraries.

## v6.0.0-dev

- Added encrypted MTProto packet building and parsing.
- Added `help.getConfig` encrypted example.
- Added gzip-packed RPC result support.

## v5.0.0-dev

- Added pure Go MTProto authorization key generation.
- Added AES-IGE and RSA_PAD support.

## v4.0.0-dev

- Clean fresh base with Bot API runtime and pure MTProto probe.
