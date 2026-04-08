---
description: YAGNI principle enforcement for lean, focused development
applies_to: ["src/**/*.ts", "src/**/*.tsx", "**/*.go"]
priority: high
---

# YAGNI Principle Best Practices

## Core YAGNI Principle

**Rule:** Implement only functionality that is currently needed for requirements
**Rule:** Do not add features based on speculation about future needs
**Rule:** Do not create abstractions until multiple concrete use cases exist
**Rule:** Do not build infrastructure for problems that do not yet exist
**Rule:** Every line of code must serve a current, documented requirement

## Feature Implementation

**Rule:** Implement features only when explicitly requested in current requirements
**Rule:** Never add "nice to have" features without explicit approval
**Rule:** Never implement features because they "might be useful later"
**Rule:** Never add configuration options that have no current use case
**Rule:** Defer optional features until they are actually requested
**Rule:** Remove placeholder features that were never completed
**Rule:** Question any feature that lacks clear acceptance criteria

## Code Abstraction

**Rule:** Start with concrete implementations before abstracting
**Rule:** Create abstractions only after identifying three similar concrete cases
**Rule:** Avoid premature generalization of single-use code
**Rule:** Never create interfaces with only one implementation
**Rule:** Never create abstract base classes without multiple concrete subclasses
**Rule:** Avoid complex inheritance hierarchies for future extensibility
**Rule:** Simplify abstractions when use cases decrease

## Architecture Decisions

**Rule:** Choose simplest architecture that meets current requirements
**Rule:** Add architectural complexity only when simpler solutions fail
**Rule:** Never design for hypothetical scale beyond current needs
**Rule:** Never implement microservices for problems solvable with monoliths
**Rule:** Defer distributed systems until single-server solutions proven inadequate
**Rule:** Avoid overengineering for theoretical future requirements
**Rule:** Refactor to more complex patterns only when proven necessary

## Framework and Library Selection

**Rule:** Use established frameworks meeting current needs
**Rule:** Avoid adopting new technologies without clear current benefit
**Rule:** Never add dependencies for features not yet needed
**Rule:** Remove unused dependencies regularly
**Rule:** Choose minimal frameworks over feature-rich alternatives
**Rule:** Avoid frameworks with learning curves exceeding current value
**Rule:** Prefer standard library solutions before adding dependencies

## Configuration and Flexibility

**Rule:** Hard-code values until actual need for configuration arises
**Rule:** Add configuration options only when multiple environments require them
**Rule:** Never create configuration systems for single-use values
**Rule:** Remove unused configuration options
**Rule:** Avoid building generic parameter-driven systems for specific use cases
**Rule:** Simplify configuration when flexibility goes unused

## Database Design

**Rule:** Design schemas for current data requirements
**Rule:** Add indexes only when queries demonstrate performance issues
**Rule:** Never create tables for data not yet collected
**Rule:** Avoid complex normalization beyond current query patterns
**Rule:** Add database optimization only after profiling reveals bottlenecks
**Rule:** Remove unused tables and columns regularly

## API Design

**Rule:** Create API endpoints only for current client needs
**Rule:** Add API versioning only when backwards compatibility breaks
**Rule:** Never design APIs for undocumented future consumers
**Rule:** Remove deprecated endpoints after migration period
**Rule:** Start with simple request/response before adding complexity
**Rule:** Add pagination only when datasets grow beyond reasonable limits
**Rule:** Implement filtering only for filters actually used

## Testing Strategy

**Rule:** Write tests for implemented functionality only
**Rule:** Never write tests for features not yet built
**Rule:** Test actual code paths rather than theoretical scenarios
**Rule:** Remove tests when removing corresponding features
**Rule:** Focus test coverage on code that exists and runs

## Performance Optimization

**Rule:** Optimize only after profiling identifies actual bottlenecks
**Rule:** Never implement caching without measuring real performance issues
**Rule:** Avoid premature optimization based on assumptions
**Rule:** Choose readable code over optimized code until proven slow
**Rule:** Add complexity for performance only when metrics justify it
**Rule:** Remove optimization code when bottlenecks no longer exist

## Error Handling

**Rule:** Handle errors that actually occur in practice
**Rule:** Never create error handlers for theoretical edge cases
**Rule:** Add validation only for inputs that have caused problems
**Rule:** Implement retries only after failures prove transient
**Rule:** Add circuit breakers only after cascading failures occur
**Rule:** Remove unused error handling code

## Security Implementation

**Rule:** Implement security measures for actual threats only
**Rule:** Add security controls when requirements or threats identified
**Rule:** Never implement security theatre without threat model
**Rule:** Balance security complexity against actual risk
**Rule:** Remove security measures that protect against non-existent threats

## Documentation

**Rule:** Document features that exist and are used
**Rule:** Never document planned features not yet implemented
**Rule:** Remove documentation for removed features
**Rule:** Focus documentation on actual usage patterns
**Rule:** Avoid documenting theoretical use cases

## Code Comments

**Rule:** Comment only to explain non-obvious implemented logic
**Rule:** Never leave TODO comments for speculative features
**Rule:** Remove comments about future plans from code
**Rule:** Delete outdated comments immediately
**Rule:** Avoid comments explaining code that should be rewritten

## Refactoring

**Rule:** Refactor only when current code causes actual problems
**Rule:** Never refactor for theoretical future flexibility
**Rule:** Simplify code when complexity no longer justified
**Rule:** Remove abstraction layers that serve no current purpose
**Rule:** Delete code that no longer has active use cases

## Data Collection

**Rule:** Collect only data currently analyzed and used
**Rule:** Never log information "just in case" without retention policy
**Rule:** Remove analytics tracking for unused metrics
**Rule:** Avoid collecting data with no defined consumer
**Rule:** Delete historical data when no longer referenced

## User Interface

**Rule:** Build UI for features that exist
**Rule:** Never create UI placeholders for future features
**Rule:** Remove unused UI components and screens
**Rule:** Simplify navigation when sections go unused
**Rule:** Avoid building complex UI frameworks for simple needs

## Integration Points

**Rule:** Integrate with external systems only when integration required
**Rule:** Never build integration layers for potential future partners
**Rule:** Remove adapter patterns when single implementation exists
**Rule:** Avoid building plugin systems without plugins
**Rule:** Simplify integration code when fewer systems remain

## Infrastructure

**Rule:** Provision infrastructure for current load only
**Rule:** Add redundancy only after downtime causes business impact
**Rule:** Never implement disaster recovery for non-critical systems
**Rule:** Scale infrastructure based on actual growth patterns
**Rule:** Remove infrastructure that serves no current purpose

## Decision Making

**Rule:** Question any feature lacking clear current value
**Rule:** Challenge assumptions about future requirements
**Rule:** Favor simplicity when uncertain about future needs
**Rule:** Re-evaluate complex solutions as requirements clarify
**Rule:** Accept that requirements change and defer premature decisions

## Code Review Checklist

**Rule:** Verify every feature addresses current documented requirement
**Rule:** Challenge any abstraction without multiple concrete uses
**Rule:** Question configuration options with no current purpose
**Rule:** Identify and remove speculative future-proofing
**Rule:** Ensure complexity justified by current needs

## Maintenance

**Rule:** Regularly audit codebase for unused code
**Rule:** Delete features that never gained adoption
**Rule:** Simplify overly flexible systems used in single way
**Rule:** Remove abstractions that no longer have multiple implementations
**Rule:** Consolidate similar code that was prematurely separated

## Exceptions

**Rule:** Security requirements may justify proactive implementation
**Rule:** Regulatory compliance may require unused capabilities
**Rule:** Critical system stability may warrant defensive measures
**Rule:** Document all exceptions to YAGNI with clear justification

## Enforcement

**Rule:** Code reviews must verify adherence to YAGNI principles
**Rule:** Architecture reviews must challenge unnecessary complexity
**Rule:** Pull requests must justify any speculative code
**Rule:** Regular refactoring sprints must remove YAGNI violations
