---
description: General coding rules for all languages
applies_to: ["**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx", "**/*.sh", "**/*.go"]
priority: high
---

# General Coding Rules for All Languages

## Code Quality and Design Principles

**Rule:** Code duplication must be eliminated through abstraction, functions, or shared modules
**Rule:** Any logic appearing more than twice must be refactored into a reusable component
**Rule:** Constants and configuration values must be defined once in a centralized location
**Rule:** Use composition over inheritance to achieve code reuse

## SOLID Principles

**Rule:** Single Responsibility: Each class/module must have only one reason to change
**Rule:** Open/Closed: Code must be open for extension but closed for modification
**Rule:** Liskov Substitution: Derived classes must be substitutable for their base classes
**Rule:** Interface Segregation: Interfaces must be specific and not force implementations to depend on unused methods
**Rule:** Dependency Inversion: High-level modules must not depend on low-level modules; both must depend on abstractions

## Code Elegance

**Rule:** Choose the most concise solution that maintains clarity and readability
**Rule:** Avoid verbose or overly complex implementations when simpler alternatives exist
**Rule:** Remove all dead code, commented-out code, and unused imports/dependencies
**Rule:** Prefer declarative over imperative programming where appropriate
**Rule:** Use language-idiomatic patterns and constructs

## Backwards Compatibility

**Rule:** All changes must maintain backwards compatibility unless explicitly documented otherwise
**Rule:** Breaking changes must be approved and documented with migration paths
**Rule:** Deprecated features must be maintained for at least one major version cycle
**Rule:** Public APIs must follow semantic versioning principles
**Rule:** No existing features, functionality, settings, or state may be removed without explicit requirement
**Rule:** Configuration changes must provide defaults that maintain existing behavior
**Rule:** Data migrations must preserve all existing data and relationships
**Rule:** UI/UX changes must maintain or enhance existing user workflows

## Architectural Separation

**Rule:** Data operations and state management must be completely separated from visualization/UI logic
**Rule:** Business logic must be independent of presentation layer implementations
**Rule:** Infrastructure concerns must be isolated from domain logic
**Rule:** Cross-cutting concerns must be handled through aspect-oriented techniques or middleware
**Rule:** Each architectural layer must only depend on the layer directly below it
**Rule:** Presentation layer must not contain business logic or data access code
**Rule:** Domain/business layer must not have dependencies on UI or infrastructure
**Rule:** Data access layer must abstract storage implementation details

## State Management

**Rule:** State mutations must be centralized and predictable
**Rule:** State tracking must be implemented separately from state visualization
**Rule:** State changes must be atomic and reversible where applicable
**Rule:** Use immutable data structures for state representation

## Design Patterns

**Rule:** Use established design patterns appropriately for common problems
**Rule:** Factory patterns must be used for complex object creation
**Rule:** Observer/Pub-Sub patterns must be used for decoupled event handling
**Rule:** Repository patterns must be used for data access abstraction
**Rule:** Document pattern usage and rationale in code comments

## Dependency Management

**Rule:** Dependencies must flow from outer layers to inner layers (clean architecture)
**Rule:** Circular dependencies must be eliminated
**Rule:** External dependencies must be wrapped in adapters or facades
**Rule:** Dependency injection must be used over direct instantiation for testability

## Error Handling

**Rule:** All errors must be handled at the appropriate architectural layer
**Rule:** Domain errors must be distinct from infrastructure errors
**Rule:** Error messages must be meaningful and actionable
**Rule:** Failed operations must not leave the system in an inconsistent state

## Code Organization

**Rule:** Related functionality must be grouped into cohesive modules
**Rule:** Module boundaries must be clearly defined with explicit interfaces
**Rule:** Internal module implementation must be encapsulated
**Rule:** Follow domain-driven design for module organization

## Naming and Conventions

**Rule:** Names must clearly express intent and purpose
**Rule:** Consistent naming conventions must be used throughout the codebase
**Rule:** Avoid abbreviations and acronyms unless universally understood
**Rule:** Use descriptive names over comments to explain code

## Testing and Quality Assurance

**Rule:** Code must be written with testability in mind
**Rule:** Dependencies must be injectable for test isolation
**Rule:** Complex logic must be extractable into testable units
**Rule:** Side effects must be minimized and isolated
**Rule:** Critical business logic must have comprehensive test coverage
**Rule:** Public APIs must have contract tests
**Rule:** Edge cases and error paths must be tested

## Performance and Optimization

**Rule:** Choose algorithms with appropriate time and space complexity
**Rule:** Avoid premature optimization without profiling evidence
**Rule:** Cache expensive computations when appropriate
**Rule:** Profile before optimizing performance bottlenecks
**Rule:** Resources must be properly acquired and released
**Rule:** Memory leaks must be prevented through proper lifecycle management
**Rule:** Concurrent access must be properly synchronized
**Rule:** Use resource pooling for expensive resources

## Documentation and Maintainability

**Rule:** Public interfaces must be fully documented
**Rule:** Complex algorithms must include explanatory comments
**Rule:** Non-obvious design decisions must be documented
**Rule:** Keep documentation close to code and up-to-date
**Rule:** Code must be self-documenting through clear structure and naming
**Rule:** Complex conditions must be extracted into well-named functions
**Rule:** Magic numbers must be replaced with named constants
**Rule:** Refactor regularly to maintain code quality

## Security and Safety

**Rule:** Input validation must be performed at system boundaries
**Rule:** Sensitive data must be encrypted in transit and at rest
**Rule:** Authentication and authorization must be enforced consistently
**Rule:** Security vulnerabilities must be addressed immediately
**Rule:** Assume external input is malicious until validated
**Rule:** Check preconditions and postconditions in critical functions
**Rule:** Fail safely with appropriate error handling
**Rule:** Use static analysis tools to identify potential issues

## Compliance and Standards

**Rule:** Follow language-specific style guides defined in project documentation
**Rule:** Use automated formatting tools to ensure consistency
**Rule:** Code reviews must verify compliance with these rules
**Rule:** Configure linters and static analysis in CI/CD pipeline
**Rule:** All changes must be reviewed before merging
**Rule:** Breaking changes must be documented in CHANGELOG
**Rule:** Refactoring must preserve external behavior
**Rule:** Use feature flags for gradual rollout of changes

## Enforcement

**Rule:** These rules apply to all code regardless of language or framework
**Rule:** Language-specific rules supplement but do not override these general rules
**Rule:** Exceptions must be documented with clear justification
**Rule:** Code violating these rules will be rejected in review
