# Changelog

## v13.0.0-dev

- Add persistent MTProto sessions.
- Add auth key save/load.
- Add server salt, time offset, DC ID, and DC address persistence.
- Add bot session file support.
- Add user account session file support.
- Add string session export/import.
- Add optional encrypted session file support with AES-256-GCM.
- Add session clear helpers.
- Update MTProto bot updates example to use saved sessions.
- Update account login example to save/load account sessions.

## Earlier dev versions

- V12.1: Pure MTProto bot updates timeout/reply fixes.
- V12: Pure MTProto bot updates loop.
- V11: Selective cache policies.
- V10: Cache and update options.
- V9: SRP 2FA login.
- V8: User account login.
- V7.1: Bot DC migration fix.
- V7: Bot authorization over MTProto.
- V6: Encrypted help.getConfig.
- V5: Auth key generation.
- V4: Clean pure-Go base and MTProto probe.
