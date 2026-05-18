package modelcombo

type Combo struct {
	Alias       string  `json:"alias" gorm:"primaryKey"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Enabled     bool    `json:"enabled" gorm:"not null;default:true"`
	Targets     Targets `json:"targets" gorm:"foreignKey:ComboAlias;references:Alias;constraint:OnDelete:CASCADE"`
	CreatedAt   int64   `json:"created_at" gorm:"autoCreateTime:milli"`
	UpdatedAt   int64   `json:"updated_at" gorm:"autoUpdateTime:milli"`
}

func (Combo) TableName() string {
	return "model_combos"
}

type Target struct {
	ID           uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	ComboAlias   string `json:"combo_alias" gorm:"index;not null"`
	ProviderID   string `json:"provider_id" gorm:"index;not null"`
	ConnectionID string `json:"connection_id" gorm:"index;not null;default:''"`
	ModelID      string `json:"model_id" gorm:"not null"`
	Enabled      bool   `json:"enabled" gorm:"not null;default:true"`
	Priority     int    `json:"priority" gorm:"not null;default:0"`
	CreatedAt    int64  `json:"created_at" gorm:"autoCreateTime:milli"`
	UpdatedAt    int64  `json:"updated_at" gorm:"autoUpdateTime:milli"`
}

func (Target) TableName() string {
	return "model_combo_targets"
}

type Targets []Target

func (targets Targets) Len() int {
	return len(targets)
}

func (targets Targets) Less(i, j int) bool {
	if targets[i].Priority == targets[j].Priority {
		return targets[i].ID < targets[j].ID
	}
	return targets[i].Priority < targets[j].Priority
}

func (targets Targets) Swap(i, j int) {
	targets[i], targets[j] = targets[j], targets[i]
}
