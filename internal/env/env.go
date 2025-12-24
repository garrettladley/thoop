package env

type Environment string

const (
	Development Environment = "development"
	Production  Environment = "production"
)

func (e Environment) IsDevelopment() bool { return e == Development }
func (e Environment) IsProduction() bool  { return e == Production }
