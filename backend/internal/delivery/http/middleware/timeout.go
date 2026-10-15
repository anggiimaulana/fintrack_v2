package middleware

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// aiDeadlineNanos holds the current write-deadline budget for non-streaming AI
// routes, in nanoseconds. It is stored atomically so the Super Admin can retune
// it at runtime without a lock on the request path; a value <= 0 means "clear
// the deadline" (unbounded).
//
// This is the HTTP-layer half of the dynamic timeout budget: the gateway owns
// the upstream-call budget (per use case) and this owns how long the connection
// may stay open while the handler waits for that call. Both read the same
// Super-Admin settings, so a change applies end-to-end within seconds.
var aiDeadlineNanos atomic.Int64

// SetAIWriteTimeout installs the write-deadline budget for AI routes. Zero or
// negative clears the deadline (the request is bounded only by the gateway's
// own per-call timeout). Safe to call concurrently with in-flight requests.
func SetAIWriteTimeout(d time.Duration) {
	aiDeadlineNanos.Store(int64(d))
}

// AIWriteTimeout returns the currently configured budget (0 = unbounded).
func AIWriteTimeout() time.Duration {
	return time.Duration(aiDeadlineNanos.Load())
}

// NoWriteTimeout clears the server's write deadline for the current request.
//
// The global http.Server.WriteTimeout is a *whole-request* budget: it is set
// when the request line is read and applies to every write until the handler
// returns. That is fine for fast CRUD, but it silently kills any long-lived
// response that legitimately streams or thinks for a while:
//
//   - SSE streams (chat, realtime) whose lifetime is unbounded;
//   - slow AI calls (receipt vision, voice transcription) where the upstream
//     model can take tens of seconds to answer.
//
// Clearing the deadline per-route (instead of raising it globally) keeps the
// safety net for ordinary endpoints while letting these few handlers run for as
// long as the upstream needs. The request context still honours the caller's
// cancellation and any per-call timeout applied further down (e.g. the AI
// gateway's route timeout), so a runaway request is still bounded.
func NoWriteTimeout() gin.HandlerFunc {
	return func(c *gin.Context) {
		clearWriteDeadline(c)
		c.Next()
	}
}

// AIDeadline extends the current request's write deadline to the dynamically
// configured AI budget (falling back to "unbounded" when none is set). Use this
// for non-streaming AI routes (receipt vision, voice) where a finite ceiling is
// desirable; SSE routes must use NoWriteTimeout instead, since a fixed deadline
// would sever a long-lived stream.
//
// Ordering invariant (each layer outlives the one below it, so the user sees the
// server's own friendly error rather than a client-side abort):
//
//	gateway per-use-case timeout  <  this HTTP deadline  <=  browser axios budget
//	                              <  Next.js proxyTimeout
//
// The budget installed here is the client budget (which the backend derives as
// "slowest use case + margin"), so the connection stays open long enough for the
// handler to flush its response after the gateway's own call timeout fires.
func AIDeadline() gin.HandlerFunc {
	return func(c *gin.Context) {
		budget := AIWriteTimeout()
		if budget <= 0 {
			clearWriteDeadline(c)
			c.Next()
			return
		}
		rc := http.NewResponseController(c.Writer)
		_ = rc.SetWriteDeadline(time.Now().Add(budget))
		c.Next()
	}
}

// clearWriteDeadline disables the write deadline on the current response.
func clearWriteDeadline(c *gin.Context) {
	// http.NewResponseController walks the writer chain via Unwrap() until it
	// finds a writer that supports SetWriteDeadline (gin's responseWriter
	// implements Unwrap since v1.10). A zero time disables the deadline.
	rc := http.NewResponseController(c.Writer)
	// Not fatal: the writer may be a wrapper that cannot set deadlines (e.g.
	// httptest). Fall through and let the handler run normally.
	_ = rc.SetWriteDeadline(time.Time{})
}
