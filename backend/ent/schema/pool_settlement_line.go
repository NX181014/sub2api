package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
)

type PoolSettlementLine struct{ ent.Schema }

func (PoolSettlementLine) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "pool_settlement_lines"}}
}

func (PoolSettlementLine) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (PoolSettlementLine) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("settlement_id"),
		field.Int64("user_id"),
		field.String("usage_weight").MaxLen(50).Default("0"),
		field.String("usage_share").MaxLen(40).Default("0"),
		field.Int64("allocated_cost_minor").Default(0),
		field.Int64("contribution_credit_minor").Default(0),
		field.Int64("adjustment_minor").Default(0),
		field.Int64("net_amount_minor").Default(0),
		field.String("payment_status").MaxLen(20).Default("unpaid"),
	}
}

func (PoolSettlementLine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("settlement_id", "user_id").Unique(),
		index.Fields("user_id"),
	}
}
