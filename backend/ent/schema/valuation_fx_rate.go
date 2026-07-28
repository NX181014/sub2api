package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

type ValuationFXRate struct{ ent.Schema }

func (ValuationFXRate) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "valuation_fx_rates"}}
}

func (ValuationFXRate) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (ValuationFXRate) Fields() []ent.Field {
	return []ent.Field{
		field.String("base_currency").MaxLen(3).Default("USD"),
		field.String("quote_currency").MaxLen(3).Default("CNY"),
		field.String("rate").MaxLen(40),
		field.Time("effective_from").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("source").Optional().Nillable().MaxLen(100),
		field.Int64("created_by_user_id"),
	}
}

func (ValuationFXRate) Indexes() []ent.Index {
	return []ent.Index{index.Fields("base_currency", "quote_currency", "effective_from").Unique()}
}
