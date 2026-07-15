// Package mail provides a portable message model and fluent composition on top
// of pluggable delivery drivers.
//
// A Message must identify its sender, at least one To, Cc, or Bcc recipient, a
// subject, and either a text or HTML body before delivery. Mailer applies
// immutable defaults before validation, so applications can establish the
// sender once with WithDefaultFrom. New requires a non-nil Driver and panics
// when that required dependency is absent.
//
// Driver implementations live in the mailfake, maillog, mailmailgun,
// mailpostmark, mailresend, mailsendgrid, mailses, and mailsmtp subpackages.
// Mailer and the bundled drivers are safe for concurrent sends when any
// caller-provided Driver, HTTP client, writer, or callback is also safe for its
// documented use.
package mail
