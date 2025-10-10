package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/shopspring/decimal"
)

// Order holds the schema definition for the Order entity.
type Order struct {
	ent.Schema
}

func (Order) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the Order.
func (Order) Fields() []ent.Field {
	return []ent.Field{
		field.String("account").MaxLen(50),
		field.String("token").MaxLen(50),
		field.String("symbol").MaxLen(32),
		field.String("gridId").MaxLen(50).Nillable().Optional(),
		field.Int("gridNumber").Nillable().Optional(),
		field.String("gridBuyCost").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.String("strategyId").MaxLen(50),
		field.Enum("type").Values("buy", "sell"),
		field.String("price").GoType(decimal.Decimal{}),
		field.String("finalPrice").GoType(decimal.Decimal{}),
		field.String("inAmount").GoType(decimal.Decimal{}),
		field.String("outAmount").GoType(decimal.Decimal{}),
		field.Enum("status").Values("pending", "closed", "rejected"),
		field.Uint64("nonce"),
		field.String("txHash").MaxLen(100),
		field.String("reason").MaxLen(500),
		field.String("profit").GoType(decimal.Decimal{}).Nillable().Optional(),
	}
}

// Edges of the Order.
func (Order) Edges() []ent.Edge {
	return nil
}

// Indexes of the Event.
func (Order) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("strategyId"),
		index.Fields("txHash"),
		index.Fields("account"),
	}
}
