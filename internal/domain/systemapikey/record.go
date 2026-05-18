package systemapikey

type Record struct {
	ID         string `json:"id" gorm:"column:id;primaryKey"`
	Name       string `json:"name" gorm:"column:name;not null"`
	Key        string `json:"key" gorm:"column:key;not null;uniqueIndex"`
	Enabled    bool   `json:"enabled" gorm:"column:enabled;not null;default:true"`
	CreatedAt  int64  `json:"created_at" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt  int64  `json:"updated_at" gorm:"column:updated_at;not null;autoUpdateTime"`
	LastUsedAt int64  `json:"last_used_at" gorm:"column:last_used_at;not null;default:0"`
}

func (Record) TableName() string {
	return "system_api_keys"
}
