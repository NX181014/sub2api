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

type PoolSettlement struct{ ent.Schema }

func (PoolSettlement) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "pool_settlements"}}
}

func (PoolSettlement) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (PoolSettlement) Fields() []ent.Field {
	return []ent.Field{
		field.String("period_type").MaxLen(10),
		field.Time("period_start").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("period_end").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("timezone").MaxLen(50).Default("Asia/Shanghai"),
		field.String("status").MaxLen(20).Default("draft"),
		field.Int64("period_cost_minor").Default(0),
		field.Int64("carry_in_minor").Default(0),
		field.Int64("carry_out_minor").Default(0),
		field.Int64("total_cost_minor").Default(0),
		field.String("total_usage_weight").MaxLen(50).Default("0"),
		field.String("pricing_coverage").MaxLen(40).Default("1"),
		field.Int64("unpriced_usage_count").Default(0),
		field.String("fx_rate").MaxLen(40).Default("1"),
		field.String("formula_version").MaxLen(20).Default("v1"),
		field.JSON("cost_snapshot", []map[string]any{}).SchemaType(map[string]string{dialect.Postgres: "jsonb"}),
		field.Int64("generated_by_user_id"),
		field.Int64("locked_by_user_id").Optional().Nillable(),
		field.Time("locked_at").Optional().Nillable().SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (PoolSettlement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status", "period_start", "period_end"),
		index.Fields("period_start", "period_end"),
	}
}
