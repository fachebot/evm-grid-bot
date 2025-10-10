package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// Nonce holds the schema definition for the Nonce entity.
type Nonce struct {
	ent.Schema
}

func (Nonce) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.Time{},
	}
}

// Fields of the Nonce.
func (Nonce) Fields() []ent.Field {
	return []ent.Field{
		field.String("account").MaxLen(42).Unique(),
		field.Uint64("nonce"),
	}
}

// Edges of the Nonce.
func (Nonce) Edges() []ent.Edge {
	return nil
}
