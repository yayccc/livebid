# go-service template

Default lightweight template for business services. Copy and trim it when a service enters implementation.

The default shape intentionally does not include `internal/service` or `internal/dto`.
For gRPC-only services, use proto messages as boundary types and let handlers orchestrate repositories until the business logic grows enough to justify a separate service layer.
