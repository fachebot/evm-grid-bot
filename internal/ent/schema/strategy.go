package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
	"github.com/shopspring/decimal"
)

// Strategy holds the schema definition for the Strategy entity.
type Strategy struct {
	ent.Schema
}

func (Strategy) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the Strategy.
func (Strategy) Fields() []ent.Field {
	return []ent.Field{
		field.String("guid").MaxLen(50),
		field.Int64("userId"),
		field.String("token").MaxLen(50),
		field.String("symbol").MaxLen(32),
		field.Float("martinFactor").Min(1),
		field.Int("maxGridLimit").Min(1).Nillable().Optional(),
		field.String("takeProfitRatio").GoType(decimal.Decimal{}),
		field.String("upperPriceBound").GoType(decimal.Decimal{}),
		field.String("lowerPriceBound").GoType(decimal.Decimal{}),
		field.String("initialOrderSize").GoType(decimal.Decimal{}),
		field.String("lastKlineVolume").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.String("fiveKlineVolume").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.Int("firstOrderId").Nillable().Optional(),
		field.String("upperBoundExit").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.String("stopLossExit").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.String("takeProfitExit").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.String("globalTakeProfitRatio").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.Bool("dynamicStopLoss").Optional(),
		field.Bool("dropOn").Optional(),
		field.Int("candlesToCheck").Optional().Default(0),
		field.String("dropThreshold").GoType(decimal.Decimal{}).Nillable().Optional(),
		field.Bool("enableAutoBuy"),
		field.Bool("enableAutoSell"),
		field.Bool("enableAutoExit"),
		field.Bool("enablePushNotification"),
		field.Enum("status").Values("active", "inactive"),
		field.String("gridTrend").Nillable().Optional(),
		field.Time("lastLowerThresholdAlertTime").Nillable().Optional(),
		field.Time("lastUpperThresholdAlertTime").Nillable().Optional(),
	}
}

// Edges of the Strategy.
func (Strategy) Edges() []ent.Edge {
	return nil
}

// Indexes of the Event.
func (Strategy) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("guid").Unique(),
		index.Fields("userId"),
		index.Fields("userId", "token").Unique(),
	}
}
