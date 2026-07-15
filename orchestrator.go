package driftpin

type Orchestrator struct {
	pin     PinStore
	scanner Scanner
	core    *CoreAlgorithm
}

func NewOrchestrator(pin PinStore, scanner Scanner) *Orchestrator {
	return &Orchestrator{
		pin:     pin,
		scanner: scanner,
		core:    NewCoreAlgorithm(),
	}
}

func (o *Orchestrator) Init() error {
	return o.pin.Save(PinState{})
}

func (o *Orchestrator) Todo() (EvaluatedState, error) {
	state, err := o.pin.Load()
	if err != nil {
		return EvaluatedState{}, err
	}

	scan, err := o.scanner.Scan()
	if err != nil {
		return EvaluatedState{}, err
	}

	ctx := CoreAlgorithmContext{
		Specs:           state.Specs,
		Markers:         state.Markers,
		Links:           state.Links,
		ResolutionState: state.ResolutionState,
		Action:          TodoAction{Scan: scan},
	}

	return o.core.EvaluateState(ctx)
}

func (o *Orchestrator) Reset(markerID, specID string) (EvaluatedState, error) {
	state, err := o.pin.Load()
	if err != nil {
		return EvaluatedState{}, err
	}

	scan, err := o.scanner.Scan()
	if err != nil {
		return EvaluatedState{}, err
	}

	ctx := CoreAlgorithmContext{
		Specs:           state.Specs,
		Markers:         state.Markers,
		Links:           state.Links,
		ResolutionState: state.ResolutionState,
		Action: ResetAction{
			SpecID:   specID,
			MarkerID: markerID,
			Scan:     scan,
		},
	}

	evaluated, err := o.core.EvaluateState(ctx)
	if err != nil {
		return EvaluatedState{}, err
	}

	err = o.pin.Save(PinState{
		Specs:           evaluated.Specs,
		Markers:         evaluated.Markers,
		Links:           evaluated.Links,
		ResolutionState: evaluated.ResolutionState,
	})
	if err != nil {
		return EvaluatedState{}, err
	}

	return evaluated, nil
}
