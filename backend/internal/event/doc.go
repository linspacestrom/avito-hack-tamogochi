// Package event defines identifiers and invariants for append-only domain
// events. Streams use aggregate versions for optimistic concurrency, while
// global positions provide deterministic ordering for projection workers.
// Command identifiers make repeated command processing idempotent. Each event
// is associated with the user ID obtained from the authenticated context.
package event
