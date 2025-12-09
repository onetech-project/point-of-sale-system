# Stakeholder Technical Notes (implementation details, for architecture review)

These notes were provided by the requester and document requested technologies and constraints. They are intentionally separate from the main feature specification so the spec remains focused on WHAT the feature must do.

- Real time in-app notification: use Server-Sent Events (SSE) for pushing notifications and order-list updates to active clients.
- Real time status update: use Server-Sent Events (SSE) for order-list and status changes.
- Email notification: use Kafka for event handling to decouple payment event processing from the email delivery pipeline.

Please review these notes during architecture and implementation planning. They represent stakeholder preferences, not mandatory implementation details in the spec.
