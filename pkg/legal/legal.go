//Package legal is a  for creating legal descriptions using metes and bounds
package legal

// TODO:
// break this into multiple files

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

//Cardinal directions proceeding north counterclockwise
const (
	North Direction = iota
	NorthEast
	East
	SouthEast
	South
	SouthWest
	West
	NorthWest
)

func dirMap() map[string]Direction {
	return map[string]Direction{
		"NORTH":     North,
		"NORTHEAST": NorthEast,
		"EAST":      East,
		"SOUTHEAST": SouthEast,
		"SOUTH":     South,
		"SOUTHWEST": SouthWest,
		"WEST":      West,
		"NORTHWEST": NorthWest,
		"N":         North,
		"NE":        NorthEast,
		"E":         East,
		"SE":        SouthEast,
		"S":         South,
		"SW":        SouthWest,
		"W":         West,
		"NW":        NorthWest,
	}
}

// DirectionFromString infers a cardinal direction from the first letter of a string. The caller is expected to have provided
// minimal input validation.
func DirectionFromString(s string) (Direction, bool) {
	s = strings.TrimSpace(strings.ToUpper(s))
	m := dirMap()
	d, ok := m[s]
	if !ok {
		return 0, false
	}
	return d, true
}

// DirectionFromAngle presumes that the angle is provided in radians
func DirectionFromAngle(angle float64) Direction {
	angle = math.Mod(angle, (2 * math.Pi))
	switch {
	case angle == 0.0:
		return North
	case angle < math.Pi/4:
		return NorthEast
	case angle == math.Pi/4:
		return East
	case angle < math.Pi/2:
		return SouthEast
	case angle == math.Pi/2:
		return South
	case angle < math.Pi*3/4:
		return SouthWest
	case angle == math.Pi*3/4:
		return West
	default:
		return NorthWest // angle cannot exceed 2pi.
	}
}

//Describe returns the string representation of a direction
func (d Direction) Describe() string {
	dirNames := [8]string{"NORTH", "NORTHEAST", "EAST", "SOUTHEAST", "SOUTH", "SOUTHWEST", "WEST", "NORTHWEST"}
	return dirNames[d]
}

// Bearing is a direction of a survey. This follows the convention of representing bearings as degrees-minutes-seconds
// TODO: Make Bearing a plain float64 in radians and marshall/unmarshall into DMS at boundaries.
type Bearing struct {
	primary   Direction
	deg       int
	min       int
	sec       float64
	secondary Direction
}

// NewBearing creates a bearing from a known quadrant and angle. example: NewBearing(North,East,15,30,45)
func NewBearing(p, snd Direction, d, m int, s float64) (Bearing, error) {
	if int(p)%2 != 0 || int(snd)%2 != 0 {
		return Bearing{}, fmt.Errorf("%s - %s are not valid directions for a bearing", p.Describe(), snd.Describe())
	}
	if d < 0 || d > 90 || m < 0 || m > 60 || s < 0.0 || s > 60.0 {
		return Bearing{}, fmt.Errorf("%d %d %f is not a valid direction", d, m, s)
	}
	return Bearing{primary: p, deg: d, min: m, sec: s, secondary: snd}, nil
}

var regBearing = regexp.MustCompile(`(?P<primary>[N|S])\D*(?P<deg>\d+)[D|°](?P<min>\d+)[M|'](?P<sec>\d+\.?\d*)[S|"](?P<secondary>[E|W])`)

// Describe is a string representation of a bearing for a legal description
func (b *Bearing) Describe() string {
	return fmt.Sprintf("%s %d°%d'%.2f\" %s", b.primary.Describe(), b.deg, b.min, b.sec, b.secondary.Describe())
}

// FromAngle construct a bearing from an angle in radians
func (b *Bearing) FromAngle(theta float64) {
	if theta < 0.0 {
		theta = math.Abs(theta) + math.Pi
	}
	theta = math.Mod(theta, math.Pi*2.0)
	var primary, secondary Direction
	switch {
	case theta < math.Pi/2.0:
		primary = North
		secondary = East
	case theta < math.Pi:
		primary = South
		secondary = East
		theta = math.Pi - theta
	case theta < math.Pi*3.0/2.0:
		primary = South
		secondary = West
		theta = theta - math.Pi
	default:
		primary = North
		secondary = West
		theta = math.Pi*2.0 - theta
	}
	total := theta * 180.0 / math.Pi
	degrees := math.Floor(total)
	totalMinutes := (total - degrees) * 60.0
	minutes := math.Floor(totalMinutes)
	seconds := (totalMinutes - minutes) * 60.0
	b.primary = primary
	b.secondary = secondary
	b.deg = int(degrees)
	b.min = int(minutes)
	b.sec = seconds
}

// FromString attempts to parse a string representation of a Bearing.
func (b *Bearing) FromString(strsrc string) error {
	str := strings.ToUpper(strings.Join(strings.Fields(strsrc), "")) // preprocess for consistency. Eliminate whitespace
	subs := regBearing.FindStringSubmatch(str)
	if len(subs) != 6 {
		return fmt.Errorf("Invalid bearing string: (%v) insufficient number of matches", subs)
	}
	subs = subs[1:]
	primary, ok := DirectionFromString(subs[0])
	if !ok {
		return fmt.Errorf("Invalid primary direction: %v", subs[0])
	}
	b.primary = primary
	deg, err := strconv.Atoi(subs[1])
	if err != nil {
		return fmt.Errorf("Invalid degrees %v", subs[1])
	}
	b.deg = deg
	min, err := strconv.Atoi(subs[2])
	if err != nil {
		return fmt.Errorf("Invalid minutes %v", subs[2])
	}
	b.min = min
	sec, err := strconv.ParseFloat(subs[3], 0)
	if err != nil {
		return fmt.Errorf("Invalid seconds %v", subs[3])
	}
	b.sec = sec
	secondary, ok := DirectionFromString(subs[4])
	if !ok {
		return fmt.Errorf("Invalid secondary direction %v", subs[4])
	}
	b.secondary = secondary
	return nil
}

// ToAngle returns the angle in radians given by a bearing
func (b *Bearing) ToAngle() float64 {
	var start, rotation float64
	if b.primary == North {
		start = 0.0
	} else {
		start = 180.0
	}
	if (b.primary == North && b.secondary == East) || (b.primary == South && b.secondary == West) {
		rotation = 1.0
	} else {
		rotation = -1.0
	}
	return (start + rotation*(float64(b.deg)+float64(b.min)/60.0+b.sec/3600.0)) / 180.0 * math.Pi
}

// Mete is a boundary used in a legal description. Metes can represent very different boundary types, but they must all produce a self-description
type Mete interface {
	Describe() string
	Preamble(float64) string
	Tangent() float64
}

// LinearMete is a boundary defined by a straight line.
type LinearMete struct {
	bearing  float64
	distance float64
	unit     string
}

func NewLinearMete(angle, distance float64, unit string) LinearMete {
	return LinearMete{
		bearing:  angle,
		distance: distance,
		unit:     unit,
	}
}

//Tangent is the angle of the bearing
func (m *LinearMete) Tangent() float64 {
	return m.bearing
}

// Describe returns a snippet of a legal description for a specific bearing
func (m *LinearMete) Describe() string {
	var b Bearing
	b.FromAngle(m.bearing)
	brng := b.Describe()
	return fmt.Sprintf("%s A DISTANCE OF %.2f %s", brng, m.distance, strings.ToUpper(m.unit))
}

// Preamble takes the tangent angle of a previous mete and describes the mete with respect to the previous (ie tangential or not)
func (m *LinearMete) Preamble(prevTan float64) string {
	if prevTan == m.bearing {
		return "A POINT OF TANGENCY"
	}
	return "A POINT OF NON-TANGENCY"
}

// FromString updates a Mete from a string as output from Autocad (ie THENCE (1) North..., 1.00 feet[;| to a point...])
// this implementation is VERY specific to AutoCAD and needs to be modified to be useful otherwise
func (m *LinearMete) FromString(line string) error {
	bearingStart := strings.Index(line, ")")
	bearingEnd := strings.Index(line, ",")
	to := strings.Index(line, "to")
	if bearingEnd == -1 || bearingStart == -1 {
		return fmt.Errorf("Invalid mete description: %s", line)
	}
	var bearing Bearing
	err := bearing.FromString(line[bearingStart:bearingEnd])
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
	m.bearing = bearing.ToAngle()
	m.distance = dist
	m.unit = unit
	return nil
}

//Rotation is a direction of travel along an arc
type Rotation int

//Rotation is given a positive or negative value to allow simple derivation of tangent angles
const (
	Clockwise        Rotation = 1
	CounterClockwise Rotation = -1
)

// ArcMete is a curved boundary line. All potentially desired information such as chord length & bearing are derived form
// the arc's angle, radius and a tangent bearing, which may or may not be the bearing of the previous mete.
type ArcMete struct {
	centralAngle float64 // angle is in radians
	radius       float64
	unit         string
	tangent      float64  // this is the angle tangent to the circle at the start in the direction of trael
	dir          Rotation // this gives us direction of travel
}

// NewArcMete creates a curved mete when parameters are known to the caller.
func NewArcMete(central, rad, tan float64, unit string, rot Rotation) *ArcMete {
	return &ArcMete{
		centralAngle: central,
		radius:       rad,
		unit:         unit,
		tangent:      tan,
		dir:          rot,
	}
}

// Tangent is the direction of the arc from its beginning.
func (am *ArcMete) Tangent() float64 {
	return am.tangent
}

// ChordLength is the straight-line distance between the start and end point along an arc
func (am *ArcMete) ChordLength() float64 {
	return 2.0 * am.radius * math.Sin(am.centralAngle/2.0)
}

// ChordAngle is the angle of the chord of an arc. It is also the tangent angle of the midpoint of the arc.
func (am *ArcMete) ChordAngle() float64 {
	return am.tangent + float64(am.dir)*am.centralAngle/2.0
}

// Concavity gives the cardinal direction of an angle from the midpoint of the arc to the center of the circle
func (am *ArcMete) Concavity() Direction {
	concaveBearing := am.tangent + float64(am.dir)*am.centralAngle/2.0 + float64(am.dir)*math.Pi/4.0
	return DirectionFromAngle(concaveBearing)
}

// ArcLength is the traveled distance along the circle.
func (am *ArcMete) ArcLength() float64 {
	return am.radius * am.centralAngle
}

// Describe returns a formatted string to be used to describe a mete in a legal description.
func (am *ArcMete) Describe() string {
	direction := DirectionFromAngle(am.ChordAngle()).Describe()
	var centBear Bearing
	centBear.FromAngle(am.centralAngle)
	cent := centBear.Describe()
	arclen := am.ArcLength()
	return fmt.Sprintf("%sERLY ALONG SAID CURVE THROUGH A CENTRAL ANGLE OF %s AN ARC DISTANCE OF %.2f %s", direction, cent, arclen, am.unit)
}

// Preamble returns a formatted string which describes the mete with respect to the previous (ie, tangency and concavity)
func (am *ArcMete) Preamble(prevAngle float64) string {
	conc := am.Concavity().Describe()
	if prevAngle == am.tangent {
		return fmt.Sprintf("THE BEGINNING OF A CURVE CONCAVE %sERLY, SAID CURVE HAS A RADIUS OF %.2f %s", conc, am.radius, am.unit)
	}
	radial := am.tangent + float64(am.dir)*math.Pi/4.0 + math.Pi/2.0 // rotate 90degrees and calculate the opposite angle
	var b Bearing
	b.FromAngle(radial)
	radBear := b.Describe()
	return fmt.Sprintf("THE BEGINNING OF A NON-TANGENT CURVE CONCAVE %sERLY, SAID CURVE HAS A RADIUS OF %.2f %s, TO WHICH A RADIAL LINE BEARS %s", conc, am.radius, am.unit, radBear)
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
	Start        Direction
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
{{if eq .Commencement true}}COMMENCING {{else}}BEGINNING {{end}} AT THE {{.Start.Describe}} CORNER OF SAID LOT{{if ne .Lot ""}} {{.Lot}}{{end}}; {{$prevtan := 0.0}}{{range $i, $m := .Metes}}{{if ne $i 0}}TO {{$m.Preamble $prevtan}}; {{end}}THENCE {{$m.Describe}} {{end}}TO THE POINT OF BEGINNING, CONTAINING {{.Area}} {{.Unit}} MORE OR LESS.`
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
