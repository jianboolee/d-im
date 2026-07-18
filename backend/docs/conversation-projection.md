# Conversation projection reliability

Conversation is a per-user read model derived from Chat, membership, and Message facts. Business services do not write it directly.

## Delivery model

- The business write and its `conversation_outbox` event are committed in one MongoDB transaction.
- The gateway worker claims persisted events and applies them asynchronously.
- Failed events use persisted retry state with bounded backoff. After 20 attempts they move to `failed` for operator replay. A worker crash is recovered by reclaiming stale `processing` events.
- Message projections advance by message sequence, so replay is idempotent and out-of-order older messages cannot overwrite newer summaries.
- Completed events are retained for seven days and then removed by TTL.

MongoDB must run as a replica set. The reliable write paths use required transactions and intentionally do not fall back to sequential writes.

## Replay

Replay retained events for one Chat:

```bash
go run ./cmd/conversation-outbox-replay -config configs/config.yaml -chat-id <chat-id>
```

Replay all retained events by omitting `-chat-id`. Replay is safe because projection mutations are idempotent.

## Repair from source data

When events have expired or a full consistency repair is required, rebuild views from Chat, active group membership, and the latest Message:

```bash
go run ./cmd/conversation-projection-repair -config configs/config.yaml -chat-id <chat-id>
```

Omit `-chat-id` to repair every Chat. Existing read positions and user settings are preserved.
