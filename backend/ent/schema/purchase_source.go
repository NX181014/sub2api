package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

type PurchaseSource struct{ ent.Schema }

func (PurchaseSource) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "purchase_sources"}}
}

func (PurchaseSource) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (PurchaseSource) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").MaxLen(100).NotEmpty(),
		field.String("website_url").Optional().Nillable().MaxLen(2048),
		field.String("notes").Optional().Nillable(),
		field.Bool("active").Default(true),
	}
}

func (PurchaseSource) Indexes() []ent.Index { return []ent.Index{index.Fields("name").Unique()} }
