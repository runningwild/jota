package champ

type Ability struct {
	Name   string
	Params map[string]float64
}

type Champion struct {
	Defname string
	*ChampionDef
}

type ChampionDef struct {
	Name      string
	Abilities []Ability
}
