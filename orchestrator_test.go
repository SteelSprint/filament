package driftpin

import (
	"errors"
	"testing"
)

type fakePinStore struct {
	state   PinState
	loadErr error
	saveErr error
	saved   []PinState
}

func (f *fakePinStore) Load() (PinState, error) {
	if f.loadErr != nil {
		return PinState{}, f.loadErr
	}
	return f.state, nil
}

func (f *fakePinStore) Save(state PinState) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.state = state
	f.saved = append(f.saved, state)
	return nil
}

type fakeScanner struct {
	scan Scan
	err  error
}

func (f *fakeScanner) Scan() (Scan, error) {
	if f.err != nil {
		return Scan{}, f.err
	}
	return f.scan, nil
}

func TestOrchestratorInit(t *testing.T) {
	t.Run("init_saves_empty_state", func(t *testing.T) {
		pin := &fakePinStore{}
		scanner := &fakeScanner{}
		orch := NewOrchestrator(pin, scanner)

		err := orch.Init()
		assertNoError(t, err)

		if len(pin.saved) != 1 {
			t.Fatalf("expected 1 save, got %d", len(pin.saved))
		}
		assertPinStateEquals(t, pin.saved[0], PinState{})
	})

	t.Run("init_propagates_save_error", func(t *testing.T) {
		pin := &fakePinStore{saveErr: errors.New("write failed")}
		scanner := &fakeScanner{}
		orch := NewOrchestrator(pin, scanner)

		err := orch.Init()
		if err == nil {
			t.Fatalf("expected error from save failure")
		}
	})
}

func TestOrchestratorTodoArity(t *testing.T) {
	shapes := []struct {
		name    string
		specs   []Spec
		markers []Marker
		links   []Link
	}{
		{"0_specs_0_markers", nil, nil, nil},
		{"1_spec_0_markers", []Spec{newSpec("s1", "b1")}, nil, nil},
		{"0_specs_1_marker", nil, []Marker{newMarker("m1", "b1")}, nil},
		{"1_spec_1_marker", []Spec{newSpec("s1", "b1")}, []Marker{newMarker("m1", "b1")}, []Link{newLink("s1", "m1")}},
		{"many_specs_1_marker", []Spec{newSpec("s1", "b1")}, []Marker{newMarker("m1", "b1"), newMarker("m2", "b2")}, []Link{newLink("s1", "m1"), newLink("s1", "m2")}},
		{"1_spec_many_markers", []Spec{newSpec("s1", "b1"), newSpec("s2", "b2")}, []Marker{newMarker("m1", "b1")}, []Link{newLink("s1", "m1"), newLink("s2", "m1")}},
		{"many_specs_many_markers", []Spec{newSpec("s1", "b1"), newSpec("s2", "b2")}, []Marker{newMarker("m1", "b1"), newMarker("m2", "b2")}, []Link{newLink("s1", "m1"), newLink("s1", "m2"), newLink("s2", "m1"), newLink("s2", "m2")}},
	}

	for _, shape := range shapes {
		t.Run(shape.name+"/no_drift", func(t *testing.T) {
			pinState := PinState{
				Specs:   shape.specs,
				Markers: shape.markers,
				Links:   shape.links,
			}
			scan := newBaselineScan(shape.specs, shape.markers)
			pin := &fakePinStore{state: pinState}
			scanner := &fakeScanner{scan: scan}
			orch := NewOrchestrator(pin, scanner)

			state, err := orch.Todo()
			assertNoError(t, err)
			assertTodoCount(t, state, 0)
		})

		t.Run(shape.name+"/all_drifted", func(t *testing.T) {
			if len(shape.links) == 0 {
				return
			}
			pinState := PinState{
				Specs:   shape.specs,
				Markers: shape.markers,
				Links:   shape.links,
			}
			specOverrides := make(map[string]string)
			for _, s := range shape.specs {
				specOverrides[s.ID] = "changed_" + s.Hash
			}
			markerOverrides := make(map[string]string)
			for _, m := range shape.markers {
				markerOverrides[m.ID] = "changed_" + m.Hash
			}
			scan := newScanFromBaselines(shape.specs, shape.markers, specOverrides, markerOverrides)
			pin := &fakePinStore{state: pinState}
			scanner := &fakeScanner{scan: scan}
			orch := NewOrchestrator(pin, scanner)

			state, err := orch.Todo()
			assertNoError(t, err)
			assertTodoCount(t, state, len(shape.links))
		})
	}
}

func TestOrchestratorTodoErrorPropagation(t *testing.T) {
	t.Run("pin_load_error", func(t *testing.T) {
		pin := &fakePinStore{loadErr: ErrPinNotFound}
		scanner := &fakeScanner{}
		orch := NewOrchestrator(pin, scanner)
		_, err := orch.Todo()
		assertErrorWraps(t, err, ErrPinNotFound)
	})

	t.Run("scanner_error", func(t *testing.T) {
		pin := &fakePinStore{}
		scanner := &fakeScanner{err: errors.New("scan failed")}
		orch := NewOrchestrator(pin, scanner)
		_, err := orch.Todo()
		if err == nil {
			t.Fatalf("expected error from scanner")
		}
	})
}

func TestOrchestratorTodoDoesNotSave(t *testing.T) {
	t.Run("todo_does_not_call_save", func(t *testing.T) {
		pinState := PinState{
			Specs:   []Spec{newSpec("s1", "b1")},
			Markers: []Marker{newMarker("m1", "b1")},
			Links:   []Link{newLink("s1", "m1")},
		}
		scan := newScanFromBaselines(pinState.Specs, pinState.Markers,
			map[string]string{"s1": "changed"},
			map[string]string{"m1": "changed"})
		pin := &fakePinStore{state: pinState}
		scanner := &fakeScanner{scan: scan}
		orch := NewOrchestrator(pin, scanner)

		_, err := orch.Todo()
		assertNoError(t, err)

		if len(pin.saved) != 0 {
			t.Fatalf("todo should not save, but got %d saves", len(pin.saved))
		}
	})
}

func TestOrchestratorReset(t *testing.T) {
	t.Run("reset_nonexistent_edge_errors", func(t *testing.T) {
		pinState := PinState{
			Specs:   []Spec{newSpec("s1", "b1")},
			Markers: []Marker{newMarker("m1", "b1")},
			Links:   []Link{newLink("s1", "m1")},
		}
		scan := newBaselineScan(pinState.Specs, pinState.Markers)
		pin := &fakePinStore{state: pinState}
		scanner := &fakeScanner{scan: scan}
		orch := NewOrchestrator(pin, scanner)

		_, err := orch.Reset("nonexistent", "nonexistent")
		assertErrorWraps(t, err, ErrResetEdgeNotLinked)
	})

	t.Run("reset_existing_edge_saves_state", func(t *testing.T) {
		specs := []Spec{newSpec("s1", "b1")}
		markers := []Marker{newMarker("m1", "b1")}
		links := []Link{newLink("s1", "m1")}
		pinState := PinState{
			Specs:   specs,
			Markers: markers,
			Links:   links,
		}
		scan := newScanFromBaselines(specs, markers,
			map[string]string{"s1": "changed"},
			map[string]string{"m1": "changed"})
		pin := &fakePinStore{state: pinState}
		scanner := &fakeScanner{scan: scan}
		orch := NewOrchestrator(pin, scanner)

		state, err := orch.Reset("m1", "s1")
		assertNoError(t, err)
		assertResolutionStateCount(t, state, 0)
		assertBaselineHashes(t, state, "s1", "changed", "m1", "changed")

		if len(pin.saved) != 1 {
			t.Fatalf("expected 1 save after reset, got %d", len(pin.saved))
		}
	})

	t.Run("reset_pin_load_error", func(t *testing.T) {
		pin := &fakePinStore{loadErr: ErrPinNotFound}
		scanner := &fakeScanner{}
		orch := NewOrchestrator(pin, scanner)
		_, err := orch.Reset("m1", "s1")
		assertErrorWraps(t, err, ErrPinNotFound)
	})

	t.Run("reset_scanner_error", func(t *testing.T) {
		pin := &fakePinStore{}
		scanner := &fakeScanner{err: errors.New("scan failed")}
		orch := NewOrchestrator(pin, scanner)
		_, err := orch.Reset("m1", "s1")
		if err == nil {
			t.Fatalf("expected error from scanner")
		}
	})

	t.Run("reset_save_error", func(t *testing.T) {
		specs := []Spec{newSpec("s1", "b1")}
		markers := []Marker{newMarker("m1", "b1")}
		links := []Link{newLink("s1", "m1")}
		pinState := PinState{
			Specs:   specs,
			Markers: markers,
			Links:   links,
		}
		scan := newScanFromBaselines(specs, markers,
			map[string]string{"s1": "changed"},
			map[string]string{"m1": "changed"})
		pin := &fakePinStore{state: pinState, saveErr: errors.New("save failed")}
		scanner := &fakeScanner{scan: scan}
		orch := NewOrchestrator(pin, scanner)

		_, err := orch.Reset("m1", "s1")
		if err == nil {
			t.Fatalf("expected error from save failure")
		}
	})
}

func TestOrchestratorResetPartialCollapse(t *testing.T) {
	t.Run("reset_one_of_two_edges_saves_resolution", func(t *testing.T) {
		specs := []Spec{newSpec("s1", "bs")}
		markers := []Marker{newMarker("m1", "bm1"), newMarker("m2", "bm2")}
		links := []Link{newLink("s1", "m1"), newLink("s1", "m2")}
		pinState := PinState{
			Specs:   specs,
			Markers: markers,
			Links:   links,
		}
		scan := newScanFromBaselines(specs, markers,
			map[string]string{"s1": "cs"},
			map[string]string{"m1": "cm1", "m2": "cm2"})
		pin := &fakePinStore{state: pinState}
		scanner := &fakeScanner{scan: scan}
		orch := NewOrchestrator(pin, scanner)

		state, err := orch.Reset("m1", "s1")
		assertNoError(t, err)
		assertResolutionStateCount(t, state, 1)
		assertBaselineHashes(t, state, "s1", "bs", "m1", "cm1")
		assertBaselineHashes(t, state, "", "", "m2", "bm2")

		if len(pin.saved) != 1 {
			t.Fatalf("expected 1 save after reset, got %d", len(pin.saved))
		}
		assertPinStateEquals(t, pin.saved[0], evaluatedStateToPinState(state))
	})
}

func evaluatedStateToPinState(state EvaluatedState) PinState {
	return PinState{
		Specs:           state.Specs,
		Markers:         state.Markers,
		Links:           state.Links,
		ResolutionState: state.ResolutionState,
	}
}
