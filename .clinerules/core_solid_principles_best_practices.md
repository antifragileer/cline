---
description: SOLID principles application rules
applies_to: ["src/**/*.ts", "src/**/*.tsx"]
priority: high
---

# SOLID Principles Best Practices

## Single Responsibility Principle (SRP)

**Rule:** Each class or module must have one, and only one, reason to change
**Rule:** Design classes with single, well-defined purpose
**Rule:** Keep methods focused on single task
**Rule:** Separate domain logic from infrastructure concerns (persistence, UI, external services)
**Rule:** Separate validation logic from processing logic
**Rule:** Use dedicated mapper classes/functions for transformations between representations
**Rule:** Large classes with many methods indicate SRP violation
**Rule:** Methods with many parameters indicate SRP violation
**Rule:** Classes changing for multiple unrelated reasons indicate SRP violation

## Open/Closed Principle (OCP)

**Rule:** Software entities must be open for extension but closed for modification
**Rule:** Use interfaces and abstract classes to define stable contracts
**Rule:** Implement different algorithms as separate classes with common interface
**Rule:** Design explicit extension points for areas of potential variation
**Rule:** Allow behavior changes through configuration not code changes
**Rule:** Favor composition over inheritance for complex behavior
**Rule:** Use events and handlers to add new behaviors without modifying existing code

## Liskov Substitution Principle (LSP)

**Rule:** Subtypes must be substitutable for their base types without altering correctness
**Rule:** Derived classes must fulfill contracts of their base classes or interfaces
**Rule:** Derived classes must not strengthen preconditions
**Rule:** Derived classes must not weaken postconditions
**Rule:** Maintain invariants of the base type
**Rule:** Overridden methods should behave consistently with base class methods
**Rule:** Avoid throwing "not implemented" exceptions in interface implementations
**Rule:** Minimize runtime type checking and casting
**Rule:** Prefer composition when inheritance would lead to LSP violations

## Interface Segregation Principle (ISP)

**Rule:** Clients must not be forced to depend on interfaces they do not use
**Rule:** Design small, cohesive interfaces with related methods
**Rule:** Create interfaces based on client roles or use cases
**Rule:** Compose multiple small interfaces rather than creating large ones
**Rule:** Clients should not implement methods they don't need
**Rule:** Tailor interfaces to specific domains or bounded contexts

## Dependency Inversion Principle (DIP)

**Rule:** High-level modules must not depend on low-level modules - both depend on abstractions
**Rule:** Abstractions must not depend on details - details depend on abstractions
**Rule:** High-level modules should depend on interfaces or abstract classes
**Rule:** Inject dependencies rather than creating them directly within classes
**Rule:** Use constructor injection for required dependencies
**Rule:** Use method injection for dependencies needed only by specific methods
**Rule:** Use IoC containers or frameworks to manage dependency creation
**Rule:** Define ports (interfaces) in domain layer, implement adapters in infrastructure layer

## Application to Agentic Layers

**Rule:** Domain Layer: Entities, Value Objects, Domain Services have clear focused responsibilities
**Rule:** Application Layer: Application Services orchestrate use cases without business logic
**Rule:** Infrastructure Layer: Each component has single responsibility
**Rule:** UI Layer: Components have focused responsibilities (presentation, interaction)
**Rule:** All layers: Depend on abstractions, use dependency injection, follow SOLID principles

## Testing SOLID Code

**Rule:** Classes with single responsibilities are easier to test in isolation
**Rule:** Extensions can be tested independently without modifying existing tests
**Rule:** Tests written for base types should pass for all derived types
**Rule:** Focused interfaces lead to more targeted, simpler tests
**Rule:** Dependencies can be easily mocked or stubbed for unit testing

## Common Anti-Patterns to Avoid

**Rule:** Avoid God Classes (large classes with many responsibilities - SRP violation)
**Rule:** Avoid Shotgun Surgery (changes requiring modifications to many classes - OCP violation)
**Rule:** Avoid Refused Bequest (subclasses not using inherited methods - LSP violation)
**Rule:** Avoid Interface Pollution (large interfaces with unrelated methods - ISP violation)
**Rule:** Avoid Concrete Dependency (direct dependencies on implementations - DIP violation)
**Rule:** Avoid Leaky Abstractions (abstractions exposing implementation details)
**Rule:** Avoid Inheritance Abuse (using inheritance for code reuse rather than polymorphism)
**Rule:** Avoid Feature Envy (class using more features of another class than its own)

## Balancing SOLID with Pragmatism

**Rule:** Start simple and refactor toward SOLID as complexity grows
**Rule:** Apply SOLID principles incrementally as needs emerge
**Rule:** Appropriate abstraction level depends on stability and complexity of requirements
**Rule:** Evaluate cost/benefit of applying SOLID principles in each situation
**Rule:** Ensure team understands principles for consistency
