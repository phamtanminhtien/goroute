package driver

type Catalog struct {
	Drivers []Driver `json:"drivers"`
}

func (c Catalog) FindByID(id string) (Driver, bool) {
	for _, drv := range c.Drivers {
		if drv.ID == id {
			return drv, true
		}
	}

	return Driver{}, false
}
