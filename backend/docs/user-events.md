# User Events

This document is the consumer contract for user events published by the business system.

The user module publishes domain facts. Consumers such as IM, search, recommendation, and notification systems may subscribe to these facts and maintain their own local user mirrors.

## Subjects

```text
dsaas.user.created
dsaas.user.profile_updated
dsaas.user.status_changed
dsaas.user.deleted
```

Subjects describe what happened in the business domain, not who should consume the event.

Do not create consumer-specific subjects such as:

```text
dsaas.im.user.created
dsaas.recommendation.user.updated
```

## Event Envelope

NATS messages contain the full event envelope:

```json
{
  "id": "event_id",
  "type": "user.created",
  "subject": "dsaas.user.created",
  "aggregate_type": "user",
  "aggregate_id": "user_id",
  "occurred_at": "2026-07-04T10:00:00+08:00",
  "source": "api",
  "data": {},
  "metadata": {}
}
```

Important fields:

```text
id              Unique event id. Consumers should use it for deduplication.
type            Domain event type, for example user.created.
subject         NATS subject.
aggregate_type  Domain aggregate type. For user events this is user.
aggregate_id    User id.
occurred_at     Time when the event was created.
source          Event source service.
data            Event-specific user snapshot.
metadata        Optional context, such as creation source.
```

## user.created

Published when a user identity is created in the business system.

Creation sources include:

```text
phone login auto-registration
admin-created human user
admin-created virtual user
admin-created system user
internal user service creation
```

Example:

```json
{
  "id": "01JZ7Y4X0P6KQK8TE6AK8ABCD1",
  "type": "user.created",
  "subject": "dsaas.user.created",
  "aggregate_type": "user",
  "aggregate_id": "usr_123",
  "occurred_at": "2026-07-04T10:00:00+08:00",
  "source": "api",
  "data": {
    "user_id": "usr_123",
    "nickname": "张三",
    "avatar_url": "",
    "gender": "",
    "bio": "",
    "status": "active",
    "role": "user",
    "user_type": "human",
    "is_protected": false,
    "verification_type": "",
    "created_at": "2026-07-04T10:00:00+08:00",
    "updated_at": "2026-07-04T10:00:00+08:00"
  },
  "metadata": {
    "source": "sms_login"
  }
}
```

Consumer recommendation:

```text
upsert local user mirror by data.user_id
```

## user.profile_updated

Published when user profile or identity attributes change.

Typical changed fields:

```text
nickname
avatar_url
gender
bio
role
user_type
is_protected
verification_type
```

Example:

```json
{
  "id": "01JZ7Y5BCJ4GYJ22N3S2WXYZ02",
  "type": "user.profile_updated",
  "subject": "dsaas.user.profile_updated",
  "aggregate_type": "user",
  "aggregate_id": "usr_123",
  "occurred_at": "2026-07-04T10:03:00+08:00",
  "source": "api",
  "data": {
    "user_id": "usr_123",
    "nickname": "张三三",
    "avatar_url": "https://example.com/avatar.png",
    "gender": "",
    "bio": "",
    "status": "active",
    "role": "user",
    "user_type": "human",
    "is_protected": false,
    "verification_type": "",
    "created_at": "2026-07-04T10:00:00+08:00",
    "updated_at": "2026-07-04T10:03:00+08:00",
    "changed_fields": ["nickname", "avatar_url"]
  },
  "metadata": {}
}
```

Consumer recommendation:

```text
Use changed_fields for optimization only.
The data object is still a snapshot. Consumers should be able to overwrite their local mirror from data.
```

## user.status_changed

Published when user status changes.

Example:

```json
{
  "id": "01JZ7Y5T7CND9M9MS0YF8WXYZ3",
  "type": "user.status_changed",
  "subject": "dsaas.user.status_changed",
  "aggregate_type": "user",
  "aggregate_id": "usr_123",
  "occurred_at": "2026-07-04T10:05:00+08:00",
  "source": "api",
  "data": {
    "user_id": "usr_123",
    "nickname": "张三三",
    "avatar_url": "https://example.com/avatar.png",
    "gender": "",
    "bio": "",
    "status": "disabled",
    "role": "user",
    "user_type": "human",
    "is_protected": false,
    "verification_type": "",
    "created_at": "2026-07-04T10:00:00+08:00",
    "updated_at": "2026-07-04T10:05:00+08:00",
    "from_status": "active",
    "to_status": "disabled",
    "reason": "manual_admin_action"
  },
  "metadata": {}
}
```

Consumer recommendation:

```text
Update the local mirror status.
Whether to stop sessions, hide content, or disable downstream behavior is decided by each consumer system.
```

## user.deleted

Published when a user is deleted in the business system.

The current business system uses soft delete for users. This event means the user should be treated as deleted or unavailable by consumers. It does not require consumers to physically delete their own records.

Example:

```json
{
  "id": "01JZ7Y69WXW9TJCGQ5WV8WXYZ4",
  "type": "user.deleted",
  "subject": "dsaas.user.deleted",
  "aggregate_type": "user",
  "aggregate_id": "usr_123",
  "occurred_at": "2026-07-04T10:08:00+08:00",
  "source": "api",
  "data": {
    "user_id": "usr_123",
    "nickname": "张三三",
    "avatar_url": "https://example.com/avatar.png",
    "gender": "",
    "bio": "",
    "status": "disabled",
    "role": "user",
    "user_type": "human",
    "is_protected": false,
    "verification_type": "",
    "created_at": "2026-07-04T10:00:00+08:00",
    "updated_at": "2026-07-04T10:05:00+08:00",
    "deleted_at": "2026-07-04T10:08:00+08:00"
  },
  "metadata": {
    "operator_id": "admin_123"
  }
}
```

Consumer recommendation:

```text
Mark the local user mirror as deleted.
Keep historical records when they are needed for audit, messages, orders, or content attribution.
Avoid hard deleting records that may still be referenced by local domain data.
```

## Data Contract

Current user snapshot fields:

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
```

Additional fields by event:

```text
user.profile_updated:
changed_fields

user.status_changed:
from_status
to_status
reason

user.deleted:
deleted_at
```

Sensitive fields are intentionally excluded:

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

## Consumer Rules

Consumers should follow these rules:

```text
Subscribe freely. The business system does not need to know every consumer.
Treat events as facts, not commands.
Use idempotent writes. Events may be delivered more than once.
Deduplicate by event id when possible.
Do not assume global ordering across subjects.
Prefer upsert by data.user_id.
Do not query the business database directly.
Keep a local mirror if user data is needed at runtime.
Use user_type to distinguish human, virtual, and system identities.
Use is_protected only as identity metadata unless the consumer has its own rule.
```

## IM Consumer Example

IM may subscribe to:

```text
dsaas.user.created
dsaas.user.profile_updated
dsaas.user.status_changed
dsaas.user.deleted
```

Suggested behavior:

```text
user.created:
  upsert im_users by user_id

user.profile_updated:
  update im_users nickname, avatar_url, user_type, and related mirror fields

user.status_changed:
  update im_users status
  decide IM-specific behavior inside the IM system

user.deleted:
  mark im_users as deleted or unavailable
  keep message history references intact
```

The business system remains the source of truth for platform identities. IM owns messaging behavior and may keep its own system conversation or message templates later, but it should not become the source of truth for business users.

## Failure And Replay

Current delivery semantics:

```text
Event recording must not roll back successful business operations.
Events recorded in eventbus_events are published by the eventbus worker.
NATS delivery is at-least-once from the consumer perspective.
Consumers must tolerate duplicate events.
Manual replay tooling may be added later.
```

If a consumer loses events, it should rebuild from a snapshot API or future replay tooling instead of reading the business database directly.
