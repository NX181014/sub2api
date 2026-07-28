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

type AccountCostEntry struct{ ent.Schema }

func (AccountCostEntry) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "account_cost_entries"}}
}

func (AccountCostEntry) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (AccountCostEntry) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		field.Int64("payer_user_id"),
		field.Int64("purchase_source_id").Optional().Nillable(),
		field.String("entry_type").MaxLen(30),
		field.String("currency").MaxLen(3).Default("CNY"),
		field.String("original_amount").MaxLen(40),
		field.Int64("cny_amount_minor"),
		field.String("fx_rate").MaxLen(40).Default("1"),
		field.Time("service_start").SchemaType(map[string]string{dialect.Postgres: "date"}),
		field.Time("service_end").SchemaType(map[string]string{dialect.Postgres: "date"}),
		field.Time("warranty_end").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "date"}),
		field.Time("paid_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("order_no").Optional().Nillable().MaxLen(255),
		field.String("purchase_url").Optional().Nillable().MaxLen(2048),
		field.String("note").Optional().Nillable(),
		field.Int64("supersedes_id").Optional().Nillable(),
		field.Int64("related_account_id").Optional().Nillable(),
		field.Int64("created_by_user_id"),
	}
}

func (AccountCostEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id", "service_start", "service_end"),
		index.Fields("payer_user_id"),
		index.Fields("purchase_source_id"),
	}
}
