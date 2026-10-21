package adapters

import "github.com/fintrack/backend/internal/ingestion"

// NewDefaultRegistry instantiates and registers all standard bank and e-wallet parsers.
func NewDefaultRegistry() *ingestion.ParserRegistry {
	reg := ingestion.NewParserRegistry()
	reg.Register(NewBCAParser())
	reg.Register(NewMandiriParser())
	reg.Register(NewJagoParser())
	reg.Register(NewSeaBankParser())
	reg.Register(NewGoPayParser())
	reg.Register(NewDANAParser())
	reg.Register(NewShopeePayParser())
	reg.Register(NewGenericCSVParser())
	return reg
}
