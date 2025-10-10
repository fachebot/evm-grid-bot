package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/shopspring/decimal"
)

// Grid holds the schema definition for the Grid entity.
type Grid struct {
	ent.Schema
}

func (Grid) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the Grid.
func (Grid) Fields() []ent.Field {
	return []ent.Field{
		field.String("guid").MaxLen(50),
		field.String("account").MaxLen(50),
		field.String("token").MaxLen(50),
		field.String("symbol").MaxLen(32),
		field.String("strategyId").MaxLen(50),
		field.Int("gridNumber").Min(0),
		field.String("orderPrice").GoType(decimal.Decimal{}),
		field.String("finalPrice").GoType(decimal.Decimal{}),
		field.String("amount").GoType(decimal.Decimal{}),
		field.String("quantity").GoType(decimal.Decimal{}),
		field.Enum("status").Values("buying", "selling", "bought"),
	}
}

// Edges of the Grid.
func (Grid) Edges() []ent.Edge {
	return nil
}

// Indexes of the Event.
func (Grid) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("guid").Unique(),
		index.Fields("strategyId"),
	}
}
