package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/skreimeyer/legal/pkg/legal"
)

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
	var metes []legal.Mete
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
