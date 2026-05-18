package routing

type Target struct {
	Prefix         string
	RequestedModel string
	ProviderID     string
	ProviderName   string
	ConnectionID   string
}

type Plan struct {
	RequestedModel string
	ResponseModel  string
	Targets        []Target
	ComboAlias     string
}

func (p Plan) PrimaryTarget() Target {
	if len(p.Targets) == 0 {
		return Target{}
	}

	return p.Targets[0]
}
