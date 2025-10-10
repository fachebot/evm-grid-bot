package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// Wallet holds the schema definition for the Wallet entity.
type Wallet struct {
	ent.Schema
}

func (Wallet) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the Wallet.
func (Wallet) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("userId"),
		field.String("account").MaxLen(50),
		field.String("password").MaxLen(100),
		field.String("privateKey").MaxLen(200),
	}
}

// Edges of the Wallet.
func (Wallet) Edges() []ent.Edge {
	return nil
}

// Indexes of the Event.
func (Wallet) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("userId").Unique(),
		index.Fields("account").Unique(),
	}
}
