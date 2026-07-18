# Eventbus

## Design

The first eventbus phase uses a non-transactional outbox. Business writes are the priority: after a business operation succeeds, the service records an event in `eventbus_events`. If recording the event fails, the business result is not rolled back; the failure is logged for investigation.

This mode provides retry and visibility for events that were recorded. It does not guarantee that every successful business change has a corresponding event row. If a future workflow requires strict consistency, that workflow can move to a transactional outbox without changing the event contract.

## Delivery Semantics

- NATS publishing is handled by the eventbus worker.
- The NATS message body must be the full event envelope, not only `data`.
- Delivery is at-least-once. Consumers must deduplicate by `id`.
- Global ordering is not guaranteed. Consumers should use `occurred_at` or domain timestamps to reconcile final state.
- Failed events stop after `max_attempts`; replay tooling can be added later.

## Subject Naming

Subjects use:

```text
dim{domain}.{event}
```

Examples:

```text
dim.group.member_joined
dim.message.recalled
```

Subjects describe domain facts, not consumers. Avoid names such as `dimim.user.created`.

## Event Envelope

```json
{
  "id": "event_id",
  "type": "group.member_joined",
  "subject": "dim.group.member_joined",
  "aggregate_type": "group",
  "aggregate_id": "group_id",
  "occurred_at": "2026-07-03T12:00:00+08:00",
  "source": "api",
  "data": {},
  "metadata": {}
}
```

## Runtime Configuration

Event recording is enabled when the API has a database connection. NATS publishing is started by enabling the worker task explicitly:

```text
WORKER_ENABLED=true
WORKER_TASKS=eventbus
NATS_URL=nats://127.0.0.1:4222
```

Optional settings:

```text
NATS_USER=
NATS_PASSWORD=
NATS_PUBLISH_TIMEOUT=2s
EVENTBUS_WORKER_BATCH_SIZE=50
EVENTBUS_WORKER_INTERVAL=2s
EVENTBUS_MAX_ATTEMPTS=5
```

`eventbus` is not included in the default `WORKER_TASKS` value yet, so local development and existing worker deployments keep their current behavior until the task is added.
