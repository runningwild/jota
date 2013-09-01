package stats

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
)

const LosPlayerHorizon = 900

type Dynamic struct {
	Health float64
}

type Base struct {
	Health float64
	Mass   float64

	// Normally at 0.0, but up to 1.0 when fully cloaked.
	Cloaking float64

	// Max rate for accelerating and turning.
	Turn float64
	Acc  float64

	// Max rate of mana draining
	Rate float64

	// Ent's radius
	Size float64

	// Maximum vision distance, for technical reasons this value will always be
	// reported as LosPlayerHorizon if it ever exceeds LosPlayerHorizon.
	Vision float64
}

type DamageKind int

const (
	DamageFire DamageKind = iota
	DamageAcid
	DamageCrushing
)

type Damage struct {
	Kind DamageKind
	Amt  float64
}

type Condition interface {
	// Called any time a base stats is queried, this will modify the base stats
	// only temporarily.
	ModifyBase(base Base) Base

	// Called any time the entity with this condition takes damage.
	ModifyDamage(damage Damage) Damage

	// Run every frame, this damage is applied to the entity with this condition.
	CauseDamage() Damage
}

type inst struct {
	Base       Base
	Dynamic    Dynamic
	Conditions []Condition
}

type Inst struct {
	inst inst
}

func (s Inst) HealthCur() float64 {
	return s.inst.Dynamic.Health
}

func (s Inst) ModifyBase(base Base) Base {
	for _, condition := range s.inst.Conditions {
		base = condition.ModifyBase(base)
	}
	return base
}
func (s Inst) HealthMax() float64 {
	return s.ModifyBase(s.inst.Base).Health
}
func (s Inst) Mass() float64 {
	return s.ModifyBase(s.inst.Base).Mass
}
func (s Inst) MaxTurn() float64 {
	return s.ModifyBase(s.inst.Base).Turn
}
func (s Inst) MaxAcc() float64 {
	return s.ModifyBase(s.inst.Base).Acc
}
func (s Inst) MaxRate() float64 {
	return s.ModifyBase(s.inst.Base).Rate
}
func (s Inst) Cloaking() float64 {
	return s.ModifyBase(s.inst.Base).Cloaking
}
func (s Inst) Size() float64 {
	return s.inst.Base.Size
}
func (s Inst) Vision() float64 {
	vision := s.ModifyBase(s.inst.Base).Vision
	if vision > LosPlayerHorizon {
		return LosPlayerHorizon
	}
	return vision
}

func (s *Inst) SetHealth(health float64) {
	s.inst.Dynamic.Health = health
}
func (s *Inst) ApplyDamage(damage Damage) {
	for _, cond := range s.inst.Conditions {
		damage = cond.ModifyDamage(damage)
	}
	if damage.Amt > 0 {
		s.inst.Dynamic.Health -= damage.Amt
	}
}
func (s *Inst) ApplyCondition(condition Condition) {
	s.inst.Conditions = append(s.inst.Conditions, condition)
}
func (s *Inst) Think() {
	// Allow any conditions to apply damage
	for _, condition := range s.inst.Conditions {
		s.ApplyDamage(condition.CauseDamage())
	}
	s.inst.Conditions = s.inst.Conditions[0:0]
	s.inst.Base.Cloaking = 0.0
}

func Make(base Base) Inst {
	var s Inst
	s.inst.Base = base
	s.inst.Dynamic = Dynamic{
		Health: s.inst.Base.Health,
	}
	return s
}

// Encoding routines - only support json and gob right now

func (si Inst) MarshalJSON() ([]byte, error) {
	return json.Marshal(si.inst)
}

func (si *Inst) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &si.inst)
}

func (si Inst) GobEncode() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(si.inst)
	return buf.Bytes(), err
}

func (si *Inst) GobDecode(data []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	err := dec.Decode(&si.inst)
	return err
}
