package connection

type Record struct {
	ID                   string `json:"id" gorm:"column:id;primaryKey"`
	ProviderID           string `json:"provider_id" gorm:"column:provider_id;not null"`
	APIKey               string `json:"api_key" gorm:"column:api_key;not null;default:''"`
	AccessToken          string `json:"access_token" gorm:"column:access_token;not null;default:''"`
	RefreshToken         string `json:"refresh_token" gorm:"column:refresh_token;not null;default:''"`
	TokenType            string `json:"token_type" gorm:"column:token_type;not null;default:''"`
	ExpiresIn            int    `json:"expires_in" gorm:"column:expires_in;not null;default:0"`
	AccessTokenExpiresAt int64  `json:"access_token_expires_at" gorm:"column:access_token_expires_at;not null;default:0"`
	Name                 string `json:"name" gorm:"column:name;not null"`
	CreatedAt            int64  `json:"-" gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt            int64  `json:"-" gorm:"column:updated_at;not null;autoUpdateTime"`
}

func (Record) TableName() string {
	return "connections"
}
