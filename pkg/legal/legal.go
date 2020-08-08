//Package legal is a  for creating legal descriptions using metes and bounds
package legal

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

//Direction is an enumeration of cardinal directions
type Direction int

//Cardinal directions
const (
	North Direction = iota
	East
	South
	West
)

var dirNames = [4]string{"NORTH", "EAST", "SOUTH", "WEST"}
var dirMap = map[byte]Direction{
	'N': North,
	'E': East,
	'S': South,
	'W': West,
}

// DirectionFromString infers a cardinal direction from the first letter of a string. The caller is expected to have provided
// minimal input validation.
func DirectionFromString(s string) (Direction, bool) {
	if len(s) == 0 {
		return 0, false
	}
	d, ok := dirMap[s[0]]
	if !ok {
		return 0, false
	}
	return d, true
}

func (d Direction) String() string {
	return dirNames[d]
}

// Bearing is a direction of a survey. This follows the convention of representing bearings as degrees-minutes-seconds
type Bearing struct {
	primary   Direction
	deg       int
	min       int
	sec       float64
	secondary Direction
}

// NewBearing creates a bearing from a known quadrant and angle. example: NewBearing(North,East,15,30,45)
func NewBearing(p, snd Direction, d, m int, s float64) (*Bearing, error) {
	if d < 0 || d > 90 || m < 0 || m > 60 || s < 0.0 || s > 60.0 {
		return nil, fmt.Errorf("%d %d %f is not a valid direction", d, m, s)
	}
	return &Bearing{primary: p, deg: d, min: m, sec: s, secondary: snd}, nil
}

var regBearing = regexp.MustCompile(`(?P<primary>[N|S])\D*(?P<deg>\d+)[D|°](?P<min>\d+)[M|'](?P<sec>\d+\.?\d*)[S|"](?P<secondary>[E|W])`)

func (b *Bearing) String() string {
	return fmt.Sprintf("%s %d°%d'%.2f\" %s", b.primary.String(), b.deg, b.min, b.sec, b.secondary.String())
}

// FromSubs populates the fields of a Bearing from a list of substrings taken from regexp.FindStringSubmatch
func (b *Bearing) FromSubs(subs []string) error {
	if len(subs) != 6 {
		return fmt.Errorf("Invalid bearing string: (%v) insufficient number of matches", subs)
	}
	subs = subs[1:]
	primary, ok := DirectionFromString(subs[0])
	if !ok {
		return fmt.Errorf("Invalid primary direction")
	}
	b.primary = primary
	deg, err := strconv.Atoi(subs[1])
	if err != nil {
		return fmt.Errorf("Invalid degrees")
	}
	b.deg = deg
	min, err := strconv.Atoi(subs[2])
	if err != nil {
		return fmt.Errorf("Invalid minutes")
	}
	b.min = min
	sec, err := strconv.ParseFloat(subs[3], 0)
	if err != nil {
		return fmt.Errorf("Invalid seconds")
	}
	b.sec = sec
	secondary, ok := DirectionFromString(subs[4])
	if !ok {
		return fmt.Errorf("Invalid secondary direction")
	}
	b.secondary = secondary
	return nil
}

// Parse update a Bearing from a string
func (b *Bearing) Parse(strsrc string) error {
	str := strings.ToUpper(strings.Join(strings.Fields(strsrc), "")) // preprocess for consistency. Eliminate whitespace
	subs := regBearing.FindStringSubmatch(str)
	err := b.FromSubs(subs)
	if err != nil {
		return err
	}
	return nil
}

// Mete is a boundary used in a legal description. A mete must be able to 1) describe itself in a canonical way and 2) create itself from a typical description
type Mete interface {
	String() string
	Parse(string) error
}

// LinearMete is a boundary defined by a straight line.
type LinearMete struct {
	bearing  Bearing
	distance float64
	unit     string
}

// String returns a snippet of a legal description for a specific bearing
func (m *LinearMete) String() string {
	return fmt.Sprintf("%s A DISTANCE OF %.2f %s", m.bearing.String(), m.distance, strings.ToUpper(m.unit))
}

// Parse updates a Mete from a string as output from Autocad (ie THENCE (1) North..., 1.00 feet[;| to a point...])
// this implementation is VERY specific to AutoCAD and needs to be modified to be useful otherwise
func (m *LinearMete) Parse(line string) error {
	bearingStart := strings.Index(line, ")")
	bearingEnd := strings.Index(line, ",")
	to := strings.Index(line, "to")
	if bearingEnd == -1 || bearingStart == -1 {
		return fmt.Errorf("Invalid mete description: %s", line)
	}
	var bearing Bearing
	err := bearing.Parse(line[bearingStart:bearingEnd])
	if err != nil {
		return err
	}
	var distSrc string
	if to == -1 {
		distSrc = line[bearingEnd+1:]
	} else {
		distSrc = line[bearingEnd:to]
	}
	distreg := regexp.MustCompile(`(\d+\.?\d*)\s?([a-zA-Z]+)`)
	results := distreg.FindStringSubmatch(distSrc)
	if len(results) < 3 {
		return fmt.Errorf("Invalid distance and units")
	}
	dist, err := strconv.ParseFloat(results[1], 64)
	if err != nil {
		return err
	}
	unit := results[2]
	m.bearing = bearing
	m.distance = dist
	m.unit = unit
	return nil
}

// ArcMete is a curved boundary line. Occasionally, the chord-angle or length is desired, but these can be derived from
// arclength and radius
type ArcMete struct {
	centralAngle Bearing
	arclength    float64
	radius       float64
	primary      Direction
	secondary    Direction
}

// NewArcMete creates a curved mete when parameters are known to the caller.
func NewArcMete(central Bearing, arclen, rad float64, primary, secondary Direction) *ArcMete {
	return &ArcMete{
		arclength: arclen,
		radius:    rad,
		primary:   primary,
		secondary: secondary,
	}
}

// ChordLength is the direct distance between the start and end point along an arc
func (am *ArcMete) ChordLength() float64 {
	angle := am.arclength / am.radius
	return 2.0 * am.radius * math.Sin(angle/2.0)
}

// Description contains all the information necessary to build a complete legal description of a bounded area
type Description struct {
	Kind         string
	Lot          string
	Block        string
	Subdivision  string
	City         string
	County       string
	State        string
	Start        string
	Commencement bool
	Area         float64
	Unit         string
	Metes        []Mete
}

// Describe creates a formatted legal description of a lot
func (d *Description) Describe() (string, error) {
	var result bytes.Buffer
	tmpl := `{{.Kind}} DESCRIPTION:

A PART OF {{if ne .Lot ""}}LOT {{.Lot}}, {{end}}{{if ne .Block ""}}BLOCK {{.Block}}, {{end}}{{.Subdivision}} TO {{if ne .City ""}}THE CITY OF {{.City}}, {{end}}{{.County}} COUNTY, {{.State}}, BEING MORE PARTICULARLY DESCRIBED AS FOLLOWS:
{{if eq .Commencement true}}COMMENCING {{else}}BEGINNING {{end}} AT THE {{.Start}} CORNER OF SAID LOT{{if ne .Lot ""}} {{.Lot}}{{end}}; {{range .Metes}}THENCE {{.String}}; {{end}}TO THE POINT OF BEGINNING, CONTAINING {{.Area}} {{.Unit}} MORE OR LESS.`
	t := template.Must(template.New("description").Parse(tmpl))
	err := t.Execute(&result, d)
	if err != nil {
		return "", err
	}
	legal := result.String()
	lastSemi := strings.LastIndex(legal, ";")
	if lastSemi != -1 {
		legal = legal[:lastSemi] + legal[lastSemi+1:]
	}
	return legal, nil
}
