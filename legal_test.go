package main

import (
	"regexp"
	"testing"
)

func cmpslice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, x := range a {
		if x != b[i] {
			return false
		}
	}
	return true
}

func TestBasic(t *testing.T) {
	ans := 2 - 4
	if ans != -2 {
		t.Errorf("IntMin(2, -2) = %d; want -2", ans)
	}
}

func TestRegex(t *testing.T) {
	re, err := regexp.Compile(`(?P<primary>[N|S|NORTH|SOUTH])(?P<deg>\d+)[d|°](?P<min>\d+)[m|'](?P<sec>\d+\.?\d*)[s|"](?P<secondary>[E|W|EAST|WEST])`)
	if err != nil {
		t.Errorf("Did not compile regex")
	}
	sample := "N10d15m30sW"
	subs := re.FindStringSubmatch(sample)[1:]
	want := []string{"N", "10", "15", "30", "W"}
	if !cmpslice(want, subs) {
		t.Errorf("Regex\nexpected:%v\nresult:%v", want, subs)
	}
}

func TestBearingFromSubs(t *testing.T) {
	re, err := regexp.Compile(`(?P<primary>[N|S|NORTH|SOUTH])(?P<deg>\d+)[d|°](?P<min>\d+)[m|'](?P<sec>\d+\.?\d*)[s|"](?P<secondary>[E|W|EAST|WEST])`)
	if err != nil {
		t.Errorf("Did not compile regex")
	}
	var result Bearing
	sample := "N10d15m30sW"
	want := Bearing{
		primary:   North,
		deg:       10,
		min:       15,
		sec:       30,
		secondary: West,
	}
	subs := re.FindStringSubmatch(sample)
	err = result.FromSubs(subs)
	if err != nil {
		t.Errorf("FromSubs method returned an error %w", err)
	}
	if want != result {
		t.Errorf("FromSubs:\nexpected:%v\n\nresult:%v", want, result)
	}
}

func TestBearingParser(t *testing.T) {
	var result Bearing
	sample := "N10d15m30sW"
	want := Bearing{
		primary:   North,
		deg:       10,
		min:       15,
		sec:       30,
		secondary: West,
	}
	err := result.Parse(sample)
	if err != nil || result != want {
		t.Errorf("ParseBearing:\nexpected:%v\nresult:%v\nerror:%w", want, result, err)
	}
	wild := `South 88°21'22.1" East`
	want = Bearing{
		primary:   South,
		deg:       88,
		min:       21,
		sec:       22.1,
		secondary: East,
	}
	err = result.Parse(wild)
	if err != nil || result != want {
		t.Errorf("ParseBearing:\nexpected:%v\nresult:%v\nerror:%w", want, result, err)
	}
}

func TestMeteParser(t *testing.T) {
	var result Mete
	sample := `THENCE (1) North 1°38'38" East, 65.00 feet to a point of non-tangency;`
	want := Mete{
		Bearing: Bearing{
			primary:   North,
			deg:       1,
			min:       38,
			sec:       38,
			secondary: East,
		},
		Distance: 65.0,
		Unit:     "feet",
	}
	err := result.Parse(sample)
	if err != nil || result != want {
		t.Errorf("ParseMete:\nexpected:%v\nresult:%v\nerror:%w", want, result, err)
	}
}
