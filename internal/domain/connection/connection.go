package connection

type Record struct {
	ID                   string
	ProviderID           string
	APIKey               string
	AccessToken          string
	RefreshToken         string
	TokenType            string
	ExpiresIn            int
	AccessTokenExpiresAt int64
	Name                 string
}
