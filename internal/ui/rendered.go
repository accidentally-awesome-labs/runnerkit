package ui

import "sync/atomic"

// errorRendered records, process-wide, that some Renderer already told the
// user about a failure (Renderer.Error, or a JSON payload with "ok": false).
// cmd/runnerkit consults it after Execute returns so an error is printed
// exactly once: commands that rendered their own diagnostic are not echoed
// again, and errors nobody rendered (Cobra parse errors, raw errors bubbling
// out of RunE) are never silently swallowed (P1-1).
var errorRendered atomic.Bool

// ErrorRendered reports whether an error diagnostic has been rendered by any
// Renderer in this process.
func ErrorRendered() bool { return errorRendered.Load() }

// MarkErrorRendered records that an error diagnostic reached the user.
func MarkErrorRendered() { errorRendered.Store(true) }

// ResetErrorRendered clears the flag. Intended for tests only.
func ResetErrorRendered() { errorRendered.Store(false) }
