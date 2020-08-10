package main

import (
	"math"
	"regexp"
	"testing"

	"github.com/skreimeyer/legal/pkg/legal"
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
	a := true
	if !a {
		t.Errorf("Testing is broken")
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

func TestDirection(t *testing.T) {
	sample := "southwest"
	southwest, ok := legal.DirectionFromString(sample)
	if !ok || southwest != legal.SouthWest {
		t.Errorf("failure to parse direction from string. source:%s\nwanted:%v\ngot:%v", sample, legal.SouthWest, southwest)
	}
	if legal.North.Describe() != "NORTH" {
		t.Errorf("directions have incorrect string representation. NORTH != %v", legal.North.Describe())
	}
}

func TestBearing(t *testing.T) {
	var result legal.Bearing
	sample := "N10d15m30sW"
	want, err := legal.NewBearing(legal.North, legal.West, 10, 15, 30.0)
	if err != nil {
		t.Errorf("NewBearing returned an error %w", err)
	}
	err = result.FromString(sample)
	if err != nil {
		t.Errorf("FromString method returned an error %w", err)
	}
	if want != result {
		t.Errorf("FromSubs:\nexpected:%v\n\nresult:%v", want, result)
	}
	var morecomplex legal.Bearing
	complex := `South 87°30'54" East`
	want, err = legal.NewBearing(legal.South, legal.East, 87, 30, 54.0)
	if err != nil {
		t.Errorf("NewBearing returned an error %w", err)
	}
	err = morecomplex.FromString(complex)
	if err != nil || morecomplex != want {
		t.Errorf("Bearing from string for %s failed with error %w and result %v", complex, err, morecomplex)
	}
}

func TestLinearFromString(t *testing.T) {
	angle := (30.0 + 1.0/60.0 + 1.0/3600.0) * math.Pi / 180.0
	want := legal.NewLinearMete(angle, 25.0, "feet")
	var result legal.LinearMete
	err := result.FromString(`THENCE (6) North 30°1'1" East, 25.00 feet`)
	if err != nil || want != result {
		t.Errorf("Linear Mete from string failed for case %v and result %v and error %w", want, result, err)
	}
}

func TestToAngle(t *testing.T) {
	epsilon := 1e-6
	bearing, err := legal.NewBearing(legal.South, legal.East, 45, 5, 5.0)
	if err != nil {
		t.Error(err)
	}
	actual := (math.Pi * 3.0 / 4.0) - (5.0/60.0+5.0/3600.0)*math.Pi/180.0
	angle := bearing.ToAngle()
	if math.Abs(actual-angle) >= epsilon {
		t.Errorf("ToAngle returns invalid result: Should have %v got %v", actual, angle)
	}
}

func TestFromAngle(t *testing.T) {
	bearing, err := legal.NewBearing(legal.South, legal.East, 45, 10, 10.0)
	if err != nil {
		t.Error(err)
	}
	var trial legal.Bearing
	trial.FromAngle((math.Pi * 3.0 / 4.0) - (10.0/60.0+10.0/3600.0)/180.0*math.Pi)
	if trial != bearing {
		t.Errorf("FromAngle 135d10m10s should be %v result %v", bearing, trial)
	}
}

func TestBearingRoundTrip(t *testing.T) {
	var b1, b2 legal.Bearing
	err := b1.FromString(`South 87°30'54" East, 5.00 feet`)
	if err != nil {
		t.Errorf("TestBearingRoundTrip parse string failed with %w", err)
	}
	angle := b1.ToAngle()
	b2.FromAngle(angle)
	if b1 != b2 {
		t.Errorf("Cannot round-trip bearings: Start %v -> angle -> bearing = %v", b1, b2)
	}

}

func TestDescription(t *testing.T) {
	var mete1 legal.LinearMete
	var mete2 legal.LinearMete
	var mete4 legal.LinearMete
	mete1.FromString(`THENCE (6) South 87°30'54" East, 5.00 feet`)
	mete2.FromString(`THENCE (5) North 2°02'36" East, 99.88 feet to a point of non-tangency`)
	mete3 := legal.NewArcMete(math.Pi/4.0, 25.0, math.Pi/6.0, "feet", legal.Clockwise)
	mete4.FromString(`THENCE (6) North 87°30'54" West, 5.00 feet`)
	d := legal.Description{
		Kind:         "Temporary Construction Easement",
		Lot:          "11",
		Block:        "15",
		Subdivision:  "Witt's Addition",
		City:         "North Little Rock",
		County:       "Pulaski",
		State:        "Arkansas",
		Start:        legal.NorthEast,
		Commencement: true,
		Area:         100.0,
		Unit:         "square feet",
		Metes:        []legal.Mete{&mete1, &mete2, mete3, &mete4},
	}
	result, err := d.Describe()
	want := "ALWAYS FAIL"
	if err != nil || result != want {
		t.Errorf("Describe failed with error: %w\n content:\n%s", err, result)
	}

}
