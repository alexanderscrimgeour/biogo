package ui

type EntityManager struct {
	animByID        map[int]*creatureAnim
	foodBlobsByKey  map[string]*Blob
	corpseBlobsByID map[int]*Blob
}

func (em *EntityManager) Reconcile(sim SimulationState) {

}
