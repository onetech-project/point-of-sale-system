<!--
SYNC IMPACT REPORT
==================
Version Change: 1.0.0 → 1.1.0
Change Type: MINOR - Added engineering principles section
Modified Principles: Expanded Principle VI (Simplicity & YAGNI) with explicit DRY guidance
Added Sections: Engineering Best Practices (10 core principles)
Removed Sections: None

Template Status:
✅ plan-template.md - Compatible with engineering principles
✅ spec-template.md - MVP approach aligns with principle 9
✅ tasks-template.md - Refactoring tasks supported by principle 8
✅ commands/*.md - Generic guidance maintained, no agent-specific updates needed

Follow-up TODOs: None
-->

# Point-of-Sale System Constitution

## Core Principles

### I. Microservice Autonomy

Each microservice MUST be independently deployable, testable, and maintainable. Services own their data, expose well-defined APIs, and communicate asynchronously where possible. No shared databases between services except via defined contracts.

**Rationale**: Enables team autonomy, independent scaling, technology flexibility, and fault isolation in a distributed retail environment where uptime is critical.

### II. API-First Design

All service interfaces MUST be designed and documented before implementation. API contracts include request/response schemas, error codes, versioning, and SLAs. Changes follow backward-compatibility rules or explicit versioning.

**Rationale**: Web app frontend and potential future clients (mobile, kiosk) depend on stable, predictable service contracts. Breaking changes without versioning cause system-wide failures.

### III. Test-First Development (NON-NEGOTIABLE)

Tests MUST be written before implementation code. Sequence: Write test → Verify test fails → Implement → Verify test passes → Refactor. All features require:
- Unit tests for business logic
- Contract tests for API boundaries
- Integration tests for service interactions

**Rationale**: Point-of-sale systems handle financial transactions and inventory. Test-first ensures correctness, prevents regressions, and provides safety for refactoring in a high-stakes domain.

### IV. Observability & Monitoring

Every service MUST emit structured logs, metrics, and traces. Implement health checks, readiness probes, and circuit breakers. Log all business-critical operations (transactions, inventory changes, user actions).

**Rationale**: Production debugging in distributed systems is impossible without observability. Transaction reconciliation and compliance audits require complete operation history.

### V. Security by Design

Authentication and authorization MUST be enforced at service boundaries. Sensitive data (payment info, customer PII) MUST be encrypted at rest and in transit. Follow principle of least privilege. Log security events.

**Rationale**: POS systems process payment information subject to PCI-DSS compliance. Security violations result in financial liability, regulatory penalties, and customer trust loss.

### VI. Simplicity First (KISS + DRY + YAGNI)

Build only what is required now. Solutions MUST be as simple as possible without unnecessary complexity. Avoid duplication through reasonable abstraction, but do not add features or abstractions without clear current need. Prefer boring, proven technologies.

**Specific Guidelines**:
- **KISS**: Choose the simplest solution that solves the problem completely
- **DRY**: Extract shared logic into reusable functions/modules when duplication appears 2-3 times
- **YAGNI**: Reject speculative features, premature optimization, and "just in case" code

**Rationale**: Over-engineering increases development time, bug surface, and maintenance burden. Business requirements in retail change frequently; adaptability beats prediction. Simple, non-duplicated code is easier to understand, test, and modify.

## Engineering Best Practices

These practices guide daily development decisions and code review standards:

### 1. SOLID Principles (Object-Oriented Design)

When using object-oriented patterns, follow SOLID principles:
- **Single Responsibility**: Each class/module has one reason to change
- **Open/Closed**: Open for extension, closed for modification
- **Liskov Substitution**: Subtypes must be substitutable for base types
- **Interface Segregation**: Clients shouldn't depend on unused interfaces
- **Dependency Inversion**: Depend on abstractions, not concretions

**Application**: Use for domain models, service layers, and complex business logic. Enables extensibility and testability.

### 2. Test-Driven Development (TDD)

Write tests before code. Cycle: Red (failing test) → Green (passing code) → Refactor (improve design).

**Rationale**: Forces clear requirements, ensures testability, prevents over-engineering, provides regression safety.

### 3. Behavior-Driven Development (BDD)

Express system behavior from user/business perspective in human-readable language (Given-When-Then).

**Application**: Use for acceptance criteria, integration tests, and feature specifications. Bridges technical and business understanding.

### 4. Continuous Integration & Deployment (CI/CD)

All changes MUST be:
- Automatically tested on every commit
- Deployable at any time
- Safe for automated deployment pipelines
- Reversible if issues detected

**Rationale**: Enables rapid feedback, reduces integration risk, accelerates delivery.

### 5. Incremental Refactoring

Improve code quality continuously without changing external behavior. Refactor when:
- Adding features touches poorly designed code
- Tests are hard to write due to tight coupling
- Code smells detected (see principle 10)

**Rules**:
- Refactor in small, safe steps
- Keep tests passing throughout
- Document architectural refactorings in ADRs

### 6. MVP Approach (Minimum Viable Product)

Prioritize minimal solutions that provide core functional value before adding optimizations or extra features.

**Process**:
1. Identify core user value
2. Build simplest implementation
3. Validate with users/tests
4. Iterate based on feedback

**Rationale**: Validates assumptions quickly, reduces waste, delivers value incrementally.

### 7. Avoid Code Smells

Actively detect and eliminate patterns indicating poor design:
- **Long methods/classes**: Break into smaller, focused units
- **Duplicated code**: Extract into shared functions/modules
- **Large parameter lists**: Introduce parameter objects or builder patterns
- **Divergent change**: Split classes that change for multiple reasons
- **Shotgun surgery**: Consolidate related changes
- **Feature envy**: Move behavior to the class it belongs
- **Data clumps**: Group related data into objects

**Action**: Address code smells during refactoring or when touched by new features.

### 8. Design for Testability

Code MUST be easy to test in isolation:
- Inject dependencies (don't instantiate deep in methods)
- Avoid static/global state
- Keep side effects at boundaries
- Write pure functions where possible

**Rationale**: Untestable code cannot follow Test-First Development principle.

### 9. Fail Fast

Detect errors as early as possible:
- Validate inputs at API boundaries
- Use type systems to prevent invalid states
- Throw exceptions for unrecoverable errors
- Return errors for recoverable failures

**Rationale**: Early detection prevents cascading failures and simplifies debugging.

### 10. Documentation as Code

Document decisions, not obvious implementation details:
- **ADRs**: Architecture Decision Records for significant choices
- **API specs**: OpenAPI/AsyncAPI for service contracts
- **Runbooks**: Operational procedures for common scenarios
- **Code comments**: Why, not what (code shows what)

**Rationale**: Documentation that lives with code stays current and accessible.

## Architecture Constraints

### Service Communication

- **Synchronous**: REST/HTTP for request-response patterns (user-facing operations)
- **Asynchronous**: Message queues for events (inventory updates, notifications)
- **Service Discovery**: Services register/discover via environment-based configuration or service mesh
- **Timeouts**: All external calls MUST have timeouts and retry logic

### Data Management

- **Database per Service**: Each microservice owns its schema
- **Data Consistency**: Use eventual consistency with compensation patterns where appropriate
- **Transactions**: Distributed transactions via saga pattern or two-phase commit only when absolutely necessary
- **Caching**: Cache at service level; document invalidation strategy

### Frontend-Backend Contract

- **Web App**: Single-page application consuming backend APIs
- **API Gateway**: Optional gateway for routing, authentication, rate limiting
- **Error Handling**: Backend returns structured error responses; frontend displays user-friendly messages

## Development Workflow

### Feature Development Lifecycle

1. **Specification Phase**: Document user stories, acceptance criteria (BDD format), API contracts
2. **Design Phase**: Create data models, service boundaries, integration points. Document architecture decisions in ADRs for complex changes
3. **Test Phase**: Write tests first (BDD acceptance → contract → unit) following TDD cycle
4. **Implementation Phase**: Write minimal code to pass tests (MVP approach)
5. **Refactoring Phase**: Improve design while keeping tests green. Address code smells
6. **Review Phase**: Code review verifies constitution compliance, test coverage, SOLID principles
7. **Deployment Phase**: CI/CD pipeline deploys to staging → validate → production

### Code Review Requirements

- All code MUST pass automated tests and linting before review
- Reviewers MUST verify:
  - Constitution compliance (principles + best practices)
  - Test coverage ≥80%
  - No code smells or anti-patterns
  - SOLID principles followed in OOP code
  - Security best practices applied
- Architecture changes require ADR and design document review
- Breaking API changes require migration plan and version bump

### Quality Gates

- **Merge Requirement**: All tests passing, coverage ≥80%, no critical security issues, linting clean
- **Deployment Requirement**: Staging validation complete, rollback plan documented, monitoring configured
- **Production Monitoring**: Alerts configured, on-call rotation staffed, runbooks updated

## Governance

This constitution supersedes all other development practices. All pull requests, design decisions, and code reviews MUST verify compliance with these principles and best practices.

### Amendment Process

- Amendments require documented justification and team consensus
- MAJOR version increment for breaking changes (principle removal/redefinition, non-negotiable rule changes)
- MINOR version increment for new principles/sections or materially expanded guidance
- PATCH version increment for clarifications, wording fixes, non-semantic refinements
- All amendments MUST include migration plan for existing code and update Sync Impact Report

### Complexity Justification

Any violation of principles MUST be explicitly justified with:
- Specific technical constraint that prevents compliance
- Simpler alternatives considered and rejected (with reasons)
- Mitigation plan to minimize impact
- Timeline for remediation if temporary
- Documented as technical debt in project tracking

### Compliance Review

- Constitution compliance reviewed in every sprint retrospective
- Violations documented and addressed as technical debt with priority
- Development guidelines in `.specify/templates/` aligned with constitution
- Engineering best practices reinforced through pair programming and code review

**Version**: 1.1.0 | **Ratified**: 2025-11-22 | **Last Amended**: 2025-12-07
