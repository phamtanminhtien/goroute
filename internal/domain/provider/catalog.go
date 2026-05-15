package provider

type Catalog struct {
	Providers []Provider `json:"providers"`
}

func (c Catalog) FindByID(id string) (Provider, bool) {
	for _, provider := range c.Providers {
		if provider.ID == id {
			return provider, true
		}
	}

	return Provider{}, false
}
