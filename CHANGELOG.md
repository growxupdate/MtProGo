# Changelog

## v10.0.0-dev

- Added configurable update handling via `WithUpdates(true/false)`.
- Added bounded message cache with `WithMessageCacheSize(n)`.
- Added bounded peer cache with `WithPeerCacheSize(n)`.
- Added cache snapshots for RAM usage visibility.
- Added `NewBotWithOptions` for bot runtime memory/update configuration.
- Added Bot API drop-pending-updates support.
- Added low-level `updates.getDifference` request/response foundation.
- Added examples: `cache_options` and `realbot_cache`.

## v9.0.0-dev

- Added pure MTProto SRP 2FA password login.

## v8.0.0-dev

- Added pure MTProto account phone-code login.

## v7.1.0-dev

- Added MTProto bot DC migration handling.

## v7.0.0-dev

- Added pure MTProto bot authorization.

## v6.0.0-dev

- Added encrypted MTProto invoke and help.getConfig.

## v5.0.0-dev

- Added pure MTProto auth key generation.

## v4.0.0-dev

- Added clean base with Bot API runtime and pure MTProto probe.
