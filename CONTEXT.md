# EdControls CLI

Command-line interface to EdControls, a construction inspection and quality-assurance platform. Used by humans and by LLM agents (chat-tool integration) to query and manage tickets, audits, and project documents.

## Language

**Contract**:
The top-level organizational unit a user belongs to; owns projects.
_Avoid_: Client (the API's internal name), organization, account

**Project**:
A construction site or engagement, always scoped to exactly one contract.
_Avoid_: Site, database (the API's internal name)

**Ticket**:
A single issue, defect, or task raised on a project; has a status (`created`, `started`, `completed`), one responsible, and zero or more tags.
_Avoid_: Issue, snag, task

**Audit**:
A template-based inspection on a project (e.g. fire-extinguisher check) consisting of questions; distinct from a ticket.
_Avoid_: Inspection, checklist

**Responsible**:
The single person (email) accountable for resolving a ticket.
_Avoid_: Assignee, owner

**Tag**:
A free-form label on a ticket, commonly used for disciplines (e.g. "brandveiligheid"). A ticket can carry multiple tags, so per-tag counts intentionally sum to more than the ticket total.
_Avoid_: Label

**Template**:
The reusable blueprint of questions an audit is created from (e.g. "fire-extinguisher check"); every audit belongs to exactly one template.
_Avoid_: Form, checklist type

**Auditor**:
The person (email) performing an audit; the audit counterpart of a ticket's responsible.
_Avoid_: Inspector, assignee

**Overdue**:
A ticket whose due date is in the past and whose status is not `completed`. Reopened tickets with a stale completion date still count as overdue.
_Avoid_: Late, expired

**Author**:
The person who created a ticket or audit.
_Avoid_: Creator, reporter
