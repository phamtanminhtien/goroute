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

func (c Catalog) WithModelRecords(records []ModelRecord) Catalog {
	providers := make([]Provider, len(c.Providers))
	for index, provider := range c.Providers {
		provider.Models = append([]Model(nil), provider.Models...)
		providers[index] = provider
	}

	modelIndexes := make(map[string]map[string]int, len(providers))
	for providerIndex, provider := range providers {
		indexes := make(map[string]int, len(provider.Models))
		for modelIndex, model := range provider.Models {
			indexes[model.ID] = modelIndex
		}
		modelIndexes[provider.ID] = indexes
		providers[providerIndex] = provider
	}

	for _, record := range records {
		for providerIndex := range providers {
			if providers[providerIndex].ID != record.ProviderID {
				continue
			}

			model := Model{
				ID:                       record.ID,
				Name:                     record.Name,
				Description:              record.Description,
				InputPricePerMillionUSD:  record.InputPricePerMillionUSD,
				OutputPricePerMillionUSD: record.OutputPricePerMillionUSD,
			}
			if modelIndex, exists := modelIndexes[record.ProviderID][record.ID]; exists {
				providers[providerIndex].Models[modelIndex] = model
			} else {
				modelIndexes[record.ProviderID][record.ID] = len(providers[providerIndex].Models)
				providers[providerIndex].Models = append(providers[providerIndex].Models, model)
			}
			break
		}
	}

	return Catalog{Providers: providers}
}
