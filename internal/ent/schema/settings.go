package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// Settings holds the schema definition for the Settings entity.
type Settings struct {
	ent.Schema
}

func (Settings) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the Settings.
func (Settings) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("userId"),
		field.Int("slippageBps").Min(0),
		field.Int("sellSlippageBps").Min(0).Nillable().Optional(),
		field.Int("exitSlippageBps").Min(0).Nillable().Optional(),
		field.Enum("dexAggregator").Values("relay"),
		field.Bool("enableInfiniteApproval").Nillable().Optional(),
	}
}

// Edges of the Settings.
func (Settings) Edges() []ent.Edge {
	return nil
}

// Indexes of the Event.
func (Settings) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("userId").Unique(),
	}
}
