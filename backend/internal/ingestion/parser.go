package ingestion

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/fintrack/backend/internal/domain"
)

// RawStatementRecord is a standardized transaction item extracted from a statement.
type RawStatementRecord struct {
	Date            time.Time              `json:"date"`
	RawDescription  string                 `json:"raw_description"`
	Amount          float64                `json:"amount"`
	Type            domain.TransactionType `json:"type"` // income / expense
	ReferenceNumber string                 `json:"reference_number,omitempty"`
	EndingBalance   *float64               `json:"ending_balance,omitempty"`
}

// StatementParser defines the interface for parsing statement files from banks and e-wallets.
type StatementParser interface {
	InstitutionCode() string
	CanParse(filename string, header []byte) bool
	Parse(ctx context.Context, r io.Reader) ([]RawStatementRecord, error)
}

// ParserRegistry manages and selects statement parsers.
type ParserRegistry struct {
	mu      sync.RWMutex
	parsers map[string]StatementParser
}

// NewParserRegistry creates an empty parser registry.
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: make(map[string]StatementParser),
	}
}

// Register registers a new parser.
func (r *ParserRegistry) Register(p StatementParser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parsers[p.InstitutionCode()] = p
}

// Get retrieves a parser by institution code.
func (r *ParserRegistry) Get(institutionCode string) (StatementParser, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.parsers[institutionCode]
	return p, ok
}

// Detect inspects filename and header bytes to match a registered parser.
func (r *ParserRegistry) Detect(filename string, header []byte) StatementParser {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.parsers {
		if p.InstitutionCode() != "generic" && p.CanParse(filename, header) {
			return p
		}
	}
	if generic, ok := r.parsers["generic"]; ok {
		return generic
	}
	return nil
}

// List returns all registered parsers.
func (r *ParserRegistry) List() []StatementParser {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]StatementParser, 0, len(r.parsers))
	for _, p := range r.parsers {
		out = append(out, p)
	}
	return out
}
