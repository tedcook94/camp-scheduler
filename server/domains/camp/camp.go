package camp

import "github.com/uptrace/bun"

type Camp struct {
	bun.BaseModel `bun:"table:camps"`
	ID            *string `json:"id" bun:"type:uuid,pk,notnull,default:uuid_generate_v4()"`
	Name          string  `json:"name" bun:"camp_name,notnull"`
	Location      *string `json:"location" bun:"camp_location"`
	Enabled       bool    `json:"enabled" bun:"camp_enabled,notnull,default:true"`
}
