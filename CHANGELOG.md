# Changelog

## v15.0.0-dev

- Normalize high-level MTProto chat IDs to Telegram/Bot-API-style IDs.
- Private chat IDs remain positive.
- Basic group chat IDs are now negative.
- Supergroup/channel chat IDs now use the `-100...` format.
- Add `Message.RawChatID` for the original positive MTProto peer ID.
- Allow replies and SendMessage/EditMessage calls to accept normalized public chat IDs.
- Add chat ID normalization helpers and tests.

## v14.2.0-dev

- Fix current Telegram layer channel/supergroup constructor ID for peer access-hash scanning.
- Keep legacy channel constructor support for older cached/test payloads.
- Fix group/supergroup replies that previously failed with missing channel access data.


## v14.1.0-dev

- Fix MTProto group/supergroup update handling by parsing `other_updates`.
- Add `updateNewMessage` and `updateNewChannelMessage` parsing from `updates.getDifference`.
- Add `supergroup` chat type and `filters.Supergroup`.
- Make `filters.Group` match both basic groups and supergroups.
- Preserve supergroup peer type from the chats vector while replying through `inputPeerChannel`.
- Keep V14 private chat behavior unchanged.

## v14.0.0-dev

- Add MTProto PeerChat and PeerChannel parsing foundation.
- Add normalized message chat types: private, group, and channel.
- Add filters.Private, filters.Group, filters.Channel, and filters.ChatType.
- Add MTProto peer cache support for users, basic groups, and channels/supergroups.
- Add high-level MTProto SendMessage helper for cached peers.
- Add basic EditMessage and DeleteMessages helpers.
- Add Message.Edit and Message.Delete convenience methods.
- Add examples/mtproto_chat_helpers.


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
