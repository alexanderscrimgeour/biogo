package systems 

type positionEntity struct {
	*ecs.BasicEntity
	*PositionComponent
}

type PositionSystem struct {
	entities []Creature
}

func (m *PositionSystem) Add(basic *ecs.BasicEntity, position *PositionComponent) {
	m.entities = append(m.entities, positionEntity{basic, position})
}

func (m *PositionSystem) Update(dt float32) {
	for _, entity := range m.entities {
		entity.PositionComponent.Update()
	}
}