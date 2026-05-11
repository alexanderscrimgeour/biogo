package ui

type EntityManager struct {
	animByID        map[int]*creatureAnim
	foodBlobsByID   map[int]*Blob
	corpseBlobsByID map[int]*Blob
}

func (em *EntityManager) Reconcile(sim SimulationState) {

}
