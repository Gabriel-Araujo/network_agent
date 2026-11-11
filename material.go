// Package resources carries the agent's material inside the binary.
//
// It exists at the module root for one reason: a //go:embed directive cannot
// reach above its own directory, and the embedded path has to be identical to
// the mirrored path so the projection is a plain WalkDir that writes the same
// relative it read (ADR 0004). Moving resources/ into a package would have
// decoupled the two and invalidated every resources/agents/... citation in the
// docs and in the AGENT.md files the model reads.
//
// It imports nothing but embed, so no import cycle is possible: internal/config
// imports this package to project the mirror.
package resources

import "embed"

// Material is the agent's material: the Agent definitions (layers 1 to 3) and
// the reference examples. internal/config projects it into the Agent root at
// ~/.network_agent/resources/..., which is where the three readers resolve.
//
// frrouting/ is deliberately absent: it is the RAG ingester's dev-time corpus,
// passed by hand through -dir, and no turn reads it — embedding it would carry
// 9.7M into a user's home to serve nothing.
//
// Both patterns carry the all: prefix so a future dotfile under either tree is
// embedded instead of silently skipped.
//
//go:embed all:resources/agents all:resources/references
var Material embed.FS
