# Changelog

## v11.0.0-dev

- Added selective message cache policies for high-volume bots.
- Default message cache now stores only messages matching registered handlers.
- Added `WithMessageCacheMode`.
- Added `WithMessageCacheFilter`.
- Added `WithMessageCacheCommands`.
- Added cache mode to `CacheSnapshot`.
- Added `examples/selective_cache`.
- Added `examples/realbot_selective_cache`.
- Kept V11 update/cache controls and all previous MTProto milestones.
- No external Telegram client libraries.
- No external Go dependencies.
