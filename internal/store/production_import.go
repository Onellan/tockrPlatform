package store

import (
	"context"
	"crypto/ed25519"
	"time"

	"github.com/Onellan/tockrplatform/internal/platform/reconciliation"
)

// ProductionImportStore is the protected operator boundary for the durable
// Platform import. It is intentionally separate from ordinary request stores;
// consumer traffic cannot invoke production writes.
type ProductionImportStore interface {
	DryRunProductionImport(context.Context, reconciliation.SignedProductionManifest, ed25519.PublicKey, reconciliation.ProductionExecution) (reconciliation.ProductionImportReceipt, error)
	ProductionImportStatus(context.Context, string) (reconciliation.ProductionImportReceipt, error)
	ApplyProductionImport(context.Context, reconciliation.SignedProductionManifest, ed25519.PublicKey, reconciliation.ProductionExecution, time.Time, int) (reconciliation.ProductionImportReceipt, error)
	RollbackProductionImport(context.Context, reconciliation.SignedProductionManifest, ed25519.PublicKey, reconciliation.ProductionExecution, time.Time) (reconciliation.ProductionImportReceipt, error)
}
