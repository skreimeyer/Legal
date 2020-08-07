package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// Direction is an enumeration of cardinal directions
type Direction int

// Cardinal directions
const (
	North Direction = iota
	East
	South
	West
)

var dirNames = []string{"NORTH", "EAST", "SOUTH", "WEST"}
var dirMap = map[string]Direction{
	"N": North,
	"E": East,
	"S": South,
	"W": West,
}

func (d Direction) String() string {
	return dirNames[d]
}

// Bearing is a survey vector
type Bearing struct {
	primary   Direction
	deg       int
	min       int
	sec       float64
	secondary Direction
}

func (b *Bearing) String() string {
	return fmt.Sprintf("%s %d°%d'%.2f\" %s", b.primary.String(), b.deg, b.min, b.sec, b.secondary.String())
}

// FromSubs populates the fields of a Bearing from a list of substrings taken from regexp.FindStringSubmatch
func (b *Bearing) FromSubs(subs []string) error {
	if len(subs) != 6 {
		return fmt.Errorf("Invalid bearing string: (%v) insufficient number of matches", subs)
	}
	subs = subs[1:]
	primary, ok := dirMap[subs[0]]
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
	secondary, ok := dirMap[subs[4]]
	if !ok {
		return fmt.Errorf("Invalid secondary direction")
	}
	b.secondary = secondary
	return nil
}

// Parse update a Bearing from a string
func (b *Bearing) Parse(strsrc string) error {
	str := strings.ToUpper(strings.Join(strings.Fields(strsrc), ""))
	re := regexp.MustCompile(`(?P<primary>[N|S])\D*(?P<deg>\d+)[D|°](?P<min>\d+)[M|'](?P<sec>\d+\.?\d*)[S|"](?P<secondary>[E|W])`)
	subs := re.FindStringSubmatch(str)
	err := b.FromSubs(subs)
	if err != nil {
		return err
	}
	return nil
}

// LineType distinguishes a mete as either an arc or line
type LineType int

// Variants for metes
const (
	Arc LineType = iota
	Line
)

// Mete is a bearing and distance used in a legel description
type Mete struct {
	Bearing  Bearing
	Distance float64
	Unit     string
	Variant  LineType
}

// String returns a snippet of a legal description for a specific bearing
func (m *Mete) String() string {
	return fmt.Sprintf("%s A DISTANCE OF %.2f %s", m.Bearing.String(), m.Distance, strings.ToUpper(m.Unit))
}

// Parse updates a Mete from a string as output from Autocad (ie THENCE (1) North..., 1.00 feet[;| to a point...])
// this implementation is VERY specific to AutoCAD and needs to be modified to be useful otherwise
func (m *Mete) Parse(line string) error {
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
	m.Bearing = bearing
	m.Distance = dist
	m.Unit = unit
	return nil
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

func main() {
	// init flags
	usage := `legal
	
	Reads a 'metes and bounds report' from AutoCAD and prints a well-formatted legal description. Most command line flags are not optional or will not produce sensible results.
	
	basic usage:
	legal -kind="Drainage Easement" -cdir=N1d2m3sE -cdist=10.0 -lot=1 -block=1 -origin=southeast -sub="Super Great Addition" REPORTFILE.txt`
	kind := flag.String("kind", "", "Type of entity described, such as 'Temporary Construction Easement'")
	cdir := flag.String("cdir", "",
		"Bearing from point of commencement to point of beginning. Must follow the format N12d34m56sE {dir}{degree}d{minute}m{second}s{dir}")
	cdist := flag.Float64("cdist", 0.0, "Distance along 'cdir' bearing from point of commencement to point of beginning")
	lot := flag.String("lot", "", "Lot number (or letter)")
	block := flag.String("block", "", "Block number (or letter)")
	origin := flag.String("origin", "", "Cardinal direction of point of beginning or commencement of the lot being described (ie, northwest, east)")
	sub := flag.String("sub", "", "Subdivision name")
	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Println(usage)
		fmt.Println("Arguments:")
		flag.PrintDefaults()
		return
	}
	filename := flag.Args()[0]
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		return
	}
	report := string(data)
	var metes []Mete
	if *cdir != "" {
		var commBearing Bearing
		err = commBearing.Parse(*cdir)
		if err != nil {
			fmt.Println("Invalid commencement bearing")
			return
		}
		commDist := *cdist
		metes = append(metes, Mete{Bearing: commBearing, Distance: commDist, Unit: "FEET"}) // FIXME: allow other units
	}
	var area float64
	var units string
	distdir := regexp.MustCompile(`(\d+\.?\d*)\s?([A-Za-z ]+)`)
	for i, l := range strings.Split(report, "\n") {
		if i == 0 || len(l) < 1 {
			continue
		}
		if l[0] == 'T' {
			mete := Mete{}
			err = mete.Parse(l)
			if err != nil {
				fmt.Println(err)
				return
			}
			metes = append(metes, mete)
		}
		if l[0] == 'C' {
			values := distdir.FindStringSubmatch(l)
			if len(values) != 3 {
				fmt.Println("Invalid area description. Area matches:", values)
				return
			}
			area, err = strconv.ParseFloat(values[1], 64)
			if err != nil {
				fmt.Println("Invalid area description", err)
			}
			units = values[2]
		}
	}
	hasCommencement := *cdir != "" || *cdist != 0.0
	desc := Description{
		Kind:         strings.ToUpper(*kind),
		Lot:          strings.ToUpper(*lot),
		Block:        strings.ToUpper(*block),
		Subdivision:  strings.ToUpper(*sub),
		City:         "NORTH LITTLE ROCK",
		County:       "PULASKI",
		State:        "ARKANSAS",
		Start:        strings.ToUpper(*origin),
		Commencement: hasCommencement,
		Area:         area,
		Unit:         strings.ToUpper(units),
		Metes:        metes,
	}
	legal, err := desc.Describe()
	if err != nil {
		fmt.Println("Failed to generate description:%w", err)
		return
	}
	fmt.Println(legal)
	return
}
