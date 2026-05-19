# biogo

Inspired by https://github.com/davidrmiller/biosim4

A real-time artificial-life simulator written in Go. Creatures evolve over generations through a genetic system, developing neural controllers for movement, feeding, reproduction, and social behaviour in a physically simulated 2D world. Built for fun because genetic algorithms and neural networks and Golang are cool.

At the start of the simulation, if the population drops below a minimum value, a random genome is created and a creature is spawned from it. 

As time goes on, some of these random genomes will eventually create a creature that has the correct neurology to survive and reproduce. 

Early on in this simulation, you may find a situation as follows:

![Simulation early days, two clear separate strategies.](./images/generation-1.5.gif)

In the above GIF, we can see two dominant species that have evolved to monopolise separate food sources. One hardly moves, whereas the second circles rapidly. There is also a third species whose strategy is simply to zip around the world and hope for the best.

Details for a selected creature are shown on the left. While it has optimised for digesting green food, a look at its neural network reveals that it is actually surviving purely by luck. Its neurology forces it to spin in a constant circle; it simply happened to be born in a dense patch of food it can efficiently digest. Over time, as food drifts across the world, creatures that can actively sense their environment will outcompete and outlive passive organisms like this one.

## Features

### Body & Growth

- Creatures have a physical body mass that grows toward a genetically encoded target using a von Bertalanffy curve - growth is slowest at birth, fastest at roughly one-third of adult size, and tapers to zero as they approach their target mass.
- A survival mass sets a structural floor; falling below it causes death regardless of energy.
- Physical radius scales as `sqrt(mass / π)`, so larger creatures occupy more space and push smaller ones aside.

### Metabolism & Energy

- Basal metabolic rate follows Kleiber's Law (M^0.75), larger creatures burn more energy absolutely but are more efficient per unit mass.
- Three digestible food types: foliage, fungi, and meat. Each food type has its own energy density, and a creature's investment in digesting each is encoded in normalised efficiency genes (gains in one come at a cost to others).
- When energy drops below a threshold, creatures enter catabolism, breaking down structural mass at 35% efficiency to survive - less efficient than digestion, so starvation is ultimately fatal.
- gut complexity is expensive, investing in broader digestion increases metabolic cost.

### Neural Control & Learning

- Each creature runs a small feedforward neural network built from its genome.
- Weights drift toward genome-encoded instinct values at a constant rate, providing a genetic prior on behaviour.
- Hebbian learning updates weights based on co-activation, modulated by a dopamine signal the creature can self-generate via REWARD and PUNISH actions. Learning costs energy.
- Each neuron tracks its average output and adjusts its gain to maintain ~40% target activity.

### Thermoregulation

- The world has a temperature gradient along the Y-axis: cold at the top, warm at the bottom, with a flat optimal band in the middle.
- Temperatures above optimal impose a quadratic metabolic penalty
- Temperatures below optimal reduce movement speed

### Aging & Death

- Maximum lifespan scales with body size and is penalised by the MetabolicRate gene - faster-living creatures die younger
- Death occurs when energy is zero, creature is below its survival mass, or it reaches max age
- Dead creatures leave meat in the world, which decays over time.

### Food & Environment

- Food spawns from drifting fountain points, configured via the UI.
- The world maintains a target total energy - when energy falls short, the deficit is replenished via fountain spawning, preventing environmental collapse or overpopulation.
- Bite size scales with creature radius, so small creatures eat less per interaction.

### Reproduction

- Asexual reproduction (type 0): the parent splits, donating 10–50% of its mass and energy to a daughter clone, which then mutates.
- Sexual reproduction (type 1): the creature seeks a mate with genome similarity above a configurable threshold, then produces offspring via crossover.
- Tier 0–1 creatures reproduce automatically when fully energised. Tier 2+ creatures require their neural network to fire a REPRODUCE action
- There's a fallback reproduction in case creatures don't develop a NNet with REPRODUCE, this comes at an energy cost and increased mutation rate
- Offspring start with a juvenile period during which growth investment is suppressed and they can't reproduce


### Sensory System

| Tier | Sensors |
|------|---------|
| 0 | Bias, energy level, angle/distance to nearest foliage, fungi, and meat |
| 1 | Age, X/Y position, heading, speed, oscillator, blocked-forward, wall proximity |
| 2 | Food density forward, prey presence/angle, threat presence/angle, mass fraction, juvenile flag, satiation, stomach fill rate |
| 3 | Local population centre of mass, local density, local heading, kinship to nearest/local creatures, temperature, temperature delta, energy delta, random noise, food per capita |

### Action System

| Tier | Actions |
|------|---------|
| 0 | Accelerate, Rotate |
| 1 | Set oscillator period, Rest, Reproduce |
| 2 | Attack |
| 3 | Feed, Set responsiveness, Set learning rate, Reward, Punish |

---

## Genetic Evolution

### Genome Structure

Each genome is a byte array with a 19-byte header encoding physical and cognitive traits, followed by a variable-length brain section.

Header genes (one byte each, mapped linearly to their valid range):

| Gene | Effect |
|------|--------|
| BodyMass | Target adult mass |
| SurvivalMass | Structural floor; constrained to (1, BodyMass/2] |
| MetabolicRate | Scales BMR; penalises max lifespan |
| StomachSize | Digestive capacity |
| FoliageDigestionEfficiency | Normalised against fungi + meat (sum = total) |
| FungiDigestionEfficiency | Normalised, as above |
| MeatDigestionEfficiency | Normalised, as above |
| VisionRadius | Maximum sight range |
| FieldOfView | Angular width of vision cone |
| MutationRate | Scales all per-tick mutation probability |
| CognitiveBreadth | Controls which sensors/actions are accessible (tier-gated) |
| SynapticDensity | Target number of neural connections |
| Neuroplasticity | Hebbian learning rate multiplier |
| LearningThreshold | Minimum co-activation for Hebbian update |
| JuvenilePeriod | Duration of pre-reproductive development |
| ReproductionType | 0 = asexual, 1 = sexual |
| MassSplitRatio | Fraction of mass donated to offspring |
| Responsiveness | Gain on neural output to action magnitude |
| OscPeriod | Period of the internal oscillator neuron |

Brain genes are 5 bytes each: source type, source ID, sink type, sink ID, and connection weight.

### Tier System

Cognitive complexity is unlocked progressively as a lineage accumulates generations, preventing early creatures from having access to sensors and actions they have not yet evolved to use.

| Tier | Generation threshold | CognitiveBreadth range | Unlocks |
|------|----------------------|------------------------|---------|
| 0 | 0 | 0–63 | Basic movement, energy sensing, food direction |
| 1 | Configurable | 64–127 | Spatial awareness, age, speed, oscillator |
| 2 | Configurable | 128–191 | Social sensing, attack, stomach state |
| 3 | Configurable | 192–255 | Kinship, population dynamics, learning control, reward/punishment |

Mutations to CognitiveBreadth are clamped to the tier permitted by the creature's current generation count.

I'm not 100% happy with the CognitiveBreadth gene, it used to do something different (control the number of neurons) and was repurposed.

### Mutation

Each reproduction event has a chance of mutating genes via "nudging" in one direction or another. The rate of mutation itself is also genetically controlled so creatures can be more or less genetically stable. 
- When SynapticDensity increases, 75% of new brain genes are biased toward newly unlocked sensors and actions; 25% are random connections.
- Since the SynapticDensity and the CognitiveBreadth are both defined in the genome itself, and both are subject to mutation, evolving creatures may develop more or less complex brains over generations. 
- Since the Genome defines the Brain, which is itself a variable length describing a Neural network graph, the neural network may too mutate to evolve new connections, or different instictive weights.

### Crossover (sexual reproduction)

- Each of the 19 bytes is drawn independently from either parent.
- Brain: Target brain length is chosen from either parent (50/50). Overlapping genes are drawn independently from either parent; excess genes from the longer parent survive with 50% probability.
- BodyMass and SurvivalMass are inherited as a pair to preserve physical proportions.
- A post-crossover mutation pass runs at 1% probability. Currently hardcoded, not sure how I want that to work yet.

### Genome Similarity

Mate selection (sexual reproduction) and kinship sensing both compare genomes. Similarity is a normalised Hamming distance over the flat byte representation of the genome. Length differences are penalised as all-bits-different bytes, discouraging mating across large brain-size gaps.

---

## Performance Optimisations

The real meat of why I work on this project - how far can I push a golang simulation and try to fight the garbage collector? The aim is to have as many creatures, each running a neural network feedforward and doing spatial lookups, on screen as possible.

### Parallel Creature Processing

Each simulation tick splits the live creature list into N contiguous batches (N = CPU count). Each goroutine processes its batch independently, writing results to a private `pendingInstructions` buffer. After all goroutines finish, results are merged serially. 

### Spatial Hash Grid

The world is partitioned into a flat-array grid of square cells. Cell lookup is a multiply-add (`bx * numY + by`), not a hash-map probe. Separate grids exist for creatures and food. All radius queries use caller-provided pre-allocated buffers, producing zero allocations per query.

### Sensor Update Staggering

Spatial radius queries (the most expensive per-creature operation) are staggered by `creatureID mod 4`. Each creature updates its sensors every 4 ticks rather than every tick, reducing spatial query volume by ~75% with negligible behavioural impact.

### Genome Similarity Cache

Each creature carries a 64-slot direct-mapped cache for genome similarity results, keyed by `(otherID & 63, genomeUID)`. When a genome mutates it gets a new atomic UID, automatically invalidating stale entries. This avoids recomputing Hamming distance on repeated mate/kin comparisons in the same tick.

Similarity computation itself processes the genome as a flat byte array in 8-byte chunks using `bits.OnesCount64()`, reducing comparisons ~8× versus byte-by-byte.

### Deferred Command Queues

Creature actions that modify shared world state (movement, eating, attack, reproduction, death) are collected into typed queues during the parallel step, then applied serially in a fixed order (move → eat → collision → attack → feed → death → reproduce). This preserves determinism without locks.

Queues are pre-sized to population count at the start of each tick to avoid repeated allocations.

### Neural Network Layout Optimisation

After pruning disconnected neurons, the remaining edges are reordered into four contiguous partitions:

1. Sensor → Neuron
2. Neuron → Neuron
3. Sensor → Action
4. Neuron → Action

The forward-pass loop iterates each partition independently with no branch on edge type inside the loop, improving branch predictor and instruction cache behaviour.  This really stumped me in the pprof for a while. 

### Collision Detection

- Swept-sphere detection handles fast-moving creatures that would otherwise tunnel through each other.
- Only pairs where `otherID > id` are processed, halving the work.
- Collision response is mass-weighted: heavier creatures resist displacement proportionally. This is a simplified physics engine as it doesn't really take into account the force, but I found the world was very unstable when I tried so...

### Fast Approximations

- Most of the math queries on the hot path, i.e. Math.Pow(), or math.Sqrt(), or math.Tanhf() (used in sensors) have been replaced with fast approximations

### Memory Reuse

At some point I ran into allocations being the bottleneck. Therefore:

- Each creature holds pre-allocated sight buffers for foliage, fungi, meat, and creature targets. Buffers are resliced to zero and refilled each sensor tick to avoid any allocations in each tick. 
- The creature array is sparse, indexed by ID, with dead slots tracked in a free-list for O(1) reuse on spawn.
- Live creature IDs are maintained in a separate slice updated incrementally (swap-and-truncate on death), so iteration never touches dead slots.

### Display Caching

Snapshots for the UI are guarded by a dirty flag. The snapshot is only recomputed when a tick has actually advanced, and callers can provide pre-allocated output structs that the snapshot function fills in place.

The UI also has controls for simulation steps per frame - so one can run at 100x speed and reduce the overhead of draw calls in ebiten. 

### Install
`
cd biogo/
go run .
`
Parameters can be adjusted in biogo/v2/simulation/parameters.go

#### Requirements
Go 1.24.0

Dependencies:
- github.com/hajimehoshi/ebiten/v2 v2.9.9
- golang.org/x/image v0.31.0
