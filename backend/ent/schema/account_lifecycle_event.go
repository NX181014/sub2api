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

type AccountLifecycleEvent struct{ ent.Schema }

func (AccountLifecycleEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "account_lifecycle_events"}}
}

func (AccountLifecycleEvent) Mixin() []ent.Mixin { return []ent.Mixin{mixins.TimeMixin{}} }

func (AccountLifecycleEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("account_id"),
		field.String("event_type").MaxLen(30),
		field.Time("occurred_at").SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("reason").Optional().Nillable(),
		field.Int64("replacement_account_id").Optional().Nillable(),
		field.Int64("transferred_cost_minor").Default(0),
		field.String("source").MaxLen(20).Default("manual"),
		field.Int64("created_by_user_id").Optional().Nillable(),
	}
}

func (AccountLifecycleEvent) Indexes() []ent.Index {
	return []ent.Index{index.Fields("account_id", "occurred_at")}
}
