package admin

import (
	"context"

	"github.com/fintrack/backend/internal/configcatalog"
)

// CatalogAdapter bridges the configcatalog store to the admin CatalogProvider
// interface. It maps the configcatalog value types to the admin-facing DTOs so
// the admin package never imports the store's types directly (keeping the two
// layers independently testable).
type CatalogAdapter struct {
	store *configcatalog.CatalogStore
}

// NewCatalogAdapter wraps a catalog store for admin handlers.
func NewCatalogAdapter(store *configcatalog.CatalogStore) *CatalogAdapter {
	return &CatalogAdapter{store: store}
}

// IntSetting reads a setting definition's default as an int.
func (a *CatalogAdapter) IntSetting(ctx context.Context, key string, fallback int64) int64 {
	return a.store.IntSetting(ctx, key, fallback)
}

// LiveInt reads the operator's CURRENT value for an integer setting.
func (a *CatalogAdapter) LiveInt(ctx context.Context, key string, fallback int64) int64 {
	return a.store.LiveInt(ctx, key, fallback)
}

// LiveString reads the operator's CURRENT value for a string setting.
func (a *CatalogAdapter) LiveString(ctx context.Context, key, fallback string) string {
	return a.store.LiveString(ctx, key, fallback)
}

// SettingDefinition returns a setting definition in the admin DTO shape.
func (a *CatalogAdapter) SettingDefinition(ctx context.Context, key string) (SettingDef, bool) {
	d, ok := a.store.SettingDefinition(ctx, key)
	if !ok {
		return SettingDef{}, false
	}
	return SettingDef{
		Key: d.Key, Kind: d.Kind, MinValue: d.MinValue, MaxValue: d.MaxValue,
		EnumValues: d.EnumValues, MaxLen: d.MaxLen, DefaultValue: d.DefaultValue,
		GroupKey: d.GroupKey, Label: d.Label, Description: d.Description,
		IsWritable: d.IsWritable, IsSecret: d.IsSecret, SortOrder: d.SortOrder, IsActive: d.IsActive,
	}, true
}

// Definitions returns the ordered setting catalog in the admin DTO shape.
func (a *CatalogAdapter) Definitions(ctx context.Context) []SettingDef {
	src := a.store.SettingDefinitions(ctx)
	out := make([]SettingDef, 0, len(src))
	for _, d := range src {
		out = append(out, SettingDef{
			Key: d.Key, Kind: d.Kind, MinValue: d.MinValue, MaxValue: d.MaxValue,
			EnumValues: d.EnumValues, MaxLen: d.MaxLen, DefaultValue: d.DefaultValue,
			GroupKey: d.GroupKey, Label: d.Label, Description: d.Description,
			IsWritable: d.IsWritable, IsSecret: d.IsSecret, SortOrder: d.SortOrder, IsActive: d.IsActive,
		})
	}
	return out
}

// Invalidate bumps the catalog version so every instance reloads.
func (a *CatalogAdapter) Invalidate(ctx context.Context) {
	a.store.Invalidate(ctx)
}

// MetricDefinitions returns the metric catalog in the admin DTO shape.
func (a *CatalogAdapter) MetricDefinitions(ctx context.Context) []MetricDef {
	src := a.store.MetricDefinitions(ctx)
	out := make([]MetricDef, 0, len(src))
	for _, d := range src {
		out = append(out, MetricDef{
			Key: d.Key, Label: d.Label, Unit: d.Unit, Description: d.Description,
			IsEnforced: d.IsEnforced, ResolverKey: d.ResolverKey,
			RatioAdjusted: d.RatioAdjusted, PerCycle: d.PerCycle,
			GroupKey: d.GroupKey, SortOrder: d.SortOrder, IsActive: d.IsActive,
		})
	}
	return out
}

// FeatureDefinitions returns the feature catalog in the admin DTO shape.
func (a *CatalogAdapter) FeatureDefinitions(ctx context.Context) []FeatureDef {
	src := a.store.FeatureDefinitions(ctx)
	out := make([]FeatureDef, 0, len(src))
	for _, d := range src {
		out = append(out, FeatureDef{
			Key: d.Key, Label: d.Label, Description: d.Description,
			GroupKey: d.GroupKey, SortOrder: d.SortOrder, IsActive: d.IsActive,
		})
	}
	return out
}

// UpsertMetric adds or updates a metric definition.
func (a *CatalogAdapter) UpsertMetric(ctx context.Context, m MetricDef) error {
	return a.store.UpsertMetric(ctx, configcatalog.MetricDefinition{
		Key: m.Key, Label: m.Label, Unit: m.Unit, Description: m.Description,
		IsEnforced: m.IsEnforced, ResolverKey: m.ResolverKey,
		RatioAdjusted: m.RatioAdjusted, PerCycle: m.PerCycle,
		GroupKey: m.GroupKey, SortOrder: m.SortOrder,
	})
}

// DeactivateMetric soft-deletes a metric definition.
func (a *CatalogAdapter) DeactivateMetric(ctx context.Context, key string) error {
	return a.store.DeactivateMetric(ctx, key)
}

// UpsertFeature adds or updates a feature definition.
func (a *CatalogAdapter) UpsertFeature(ctx context.Context, f FeatureDef) error {
	return a.store.UpsertFeature(ctx, configcatalog.FeatureDefinition{
		Key: f.Key, Label: f.Label, Description: f.Description,
		GroupKey: f.GroupKey, SortOrder: f.SortOrder,
	})
}

// DeactivateFeature soft-deletes a feature definition.
func (a *CatalogAdapter) DeactivateFeature(ctx context.Context, key string) error {
	return a.store.DeactivateFeature(ctx, key)
}

// compile-time assertion.
var _ CatalogProvider = (*CatalogAdapter)(nil)
