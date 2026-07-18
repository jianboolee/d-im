# Conversation projection reliability

Conversation is a per-user read model derived from Chat, membership, and Message facts. Projection timing follows the consistency requirement of each use case.

## Consistency policy

- Single-chat creation synchronously projects both user Conversations in the same MongoDB transaction as the Chat.
- Group creation synchronously projects every initial member Conversation in the same MongoDB transaction as Chat, Group, and GroupMember.
- Successful Chat and Group creation responses therefore provide read-your-writes consistency.
- Message summaries and the remaining fan-out projection paths use the transactional outbox described below.
- User-owned Conversation settings and read positions are synchronous writes.

## Delivery model

- For asynchronous paths, the business write and its `conversation_outbox` event are committed in one MongoDB transaction.
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
