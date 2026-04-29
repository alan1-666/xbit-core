# GraphQL Schemas

Service schemas will live here as the Go services replace upstream-compatible facades.

The gateway currently proxies the existing frontend-compatible GraphQL endpoints. Each service should add a schema file here before implementing gqlgen resolvers.
