package champ

type Ability struct {
	Name   string
	Params map[string]int
}

type Champion struct {
	Defname string
	*ChampionDef
}

type ChampionDef struct {
	Name      string
	Abilities []Ability
}
