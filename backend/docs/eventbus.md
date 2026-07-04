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
dsaas.{domain}.{event}
```

Examples:

```text
dsaas.user.created
dsaas.user.profile_updated
dsaas.user.status_changed
dsaas.user.deleted
```

Subjects describe domain facts, not consumers. Avoid names such as `dsaas.im.user.created`.

## Event Envelope

```json
{
  "id": "event_id",
  "type": "user.created",
  "subject": "dsaas.user.created",
  "aggregate_type": "user",
  "aggregate_id": "user_id",
  "occurred_at": "2026-07-03T12:00:00+08:00",
  "source": "api",
  "data": {},
  "metadata": {}
}
```

## User Events

First phase user events:

```text
user.created
user.profile_updated
user.status_changed
user.deleted
```

User event payloads use a whitelist snapshot. They may include:

```text
user_id
nickname
avatar_url
gender
bio
status
role
user_type
is_protected
verification_type
created_at
updated_at
changed_fields
from_status
to_status
reason
deleted_at
```

They must not include:

```text
password_hash
token
openid
unionid
phone
email
identity documents
verification materials
```

## Consumer Registry

Initial expected consumers:

```text
dsaas.user.created
Consumers: im-service, recommendation-service

dsaas.user.profile_updated
Consumers: im-service, recommendation-service

dsaas.user.status_changed
Consumers: im-service, recommendation-service
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

## Current Scope

The current implementation records user module events into `eventbus_events` and can publish pending events to NATS through the eventbus worker.

Covered user module entry points:

```text
AdminCreateUser -> user.created
EnsureUserForPhoneLogin -> user.created
UpdateProfile -> user.profile_updated
AdminUpdateUser -> user.profile_updated
SetUserStatus/AdminSetStatus -> user.status_changed
AdminDeleteUser -> user.deleted
```

Known follow-up:

```text
Manual replay tooling for failed events can be added after worker behavior is observed in deployment.
```

Consumer-facing user event contract:

```text
api/docs/user-events.md
```
