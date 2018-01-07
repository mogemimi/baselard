package main

import (
	"testing"
)

func TestToString1Out1In(t *testing.T) {
	s := NinjaBuild{
		Rule:    "cc",
		Outputs: []string{"a.o"},
		Inputs:  []string{"a.c"},
	}
	actual := s.ToString()
	expected := "build a.o: cc a.c\n"
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}

func TestToString2Outs1Ins(t *testing.T) {
	s := NinjaBuild{
		Outputs: []string{"a", "b"},
		Rule:    "c",
		Inputs:  []string{"d"},
	}
	actual := s.ToString()
	expected := `build $
  a $
  b: c $
  d
`
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}

func TestToString2OutsNoIns(t *testing.T) {
	s := NinjaBuild{
		Outputs: []string{"a", "b"},
		Rule:    "c",
	}
	actual := s.ToString()
	expected := `build $
  a $
  b: c
`
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}

func TestToString1Out2Ins(t *testing.T) {
	s := NinjaBuild{
		Outputs: []string{"a"},
		Rule:    "b",
		Inputs:  []string{"c", "d"},
	}
	actual := s.ToString()
	expected := `build a: b $
  c $
  d
`
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}

func TestToStringMultipleOuts(t *testing.T) {
	s := NinjaBuild{
		Outputs: []string{"a", "b", "c"},
		Rule:    "d",
	}
	actual := s.ToString()
	expected := `build $
  a $
  b $
  c: d
`
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}

func TestToString1Out1ImplicitOut(t *testing.T) {
	s := NinjaBuild{
		Outputs:      []string{"a"},
		ImplicitOuts: []string{"b"},
		Rule:         "c",
	}
	actual := s.ToString()
	expected := `build $
  a | $
  b: c
`
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}

func TestToStringMultipleImplicitOuts(t *testing.T) {
	s := NinjaBuild{
		ImplicitOuts: []string{"a", "b"},
		Rule:         "c",
	}
	actual := s.ToString()
	expected := `build | $
  a $
  b: c
`
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}

func TestToStringMultipleImplicitDeps(t *testing.T) {
	s := NinjaBuild{
		Outputs:      []string{"a"},
		Rule:         "b",
		ImplicitDeps: []string{"c", "d"},
	}
	actual := s.ToString()
	expected := `build a: b | $
  c $
  d
`
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}

func TestToStringPool(t *testing.T) {
	s := NinjaBuild{
		Outputs:      []string{"a"},
		ImplicitOuts: []string{"b"},
		Rule:         "c",
		Inputs:       []string{"d"},
		ImplicitDeps: []string{"e"},
		Pool:         "f",
	}
	actual := s.ToString()
	expected := `build $
  a | $
  b: c $
  d | $
  e
  pool = f
`
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}

func TestToStringVariables(t *testing.T) {
	s := NinjaBuild{
		Outputs: []string{"a"},
		Rule:    "b",
		Inputs:  []string{"c"},
		Variables: map[string]string{
			"x": "1",
			"y": "2",
			"z": "3",
		},
	}
	actual := s.ToString()
	expected := `build a: b c
  x = 1
  y = 2
  z = 3
`
	if actual != expected {
		t.Errorf("Unexpected string:\n%v", actual)
	}
}
