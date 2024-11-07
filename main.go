package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"text/template"
)

type FixtureInfo struct {
	ID          string
	Description string
	VA          int
	Count       int
}

func (f *FixtureInfo) Add(qty int) {
	f.Count += qty
}

func NewFixtureInfo(id, desc string, va int) *FixtureInfo {
	return &FixtureInfo{
		ID:          id,
		Description: desc,
		VA:          va,
		Count:       1,
	}
}

var (
	digitsRegex = regexp.MustCompile(`^\d+`)
)

const (
	CCKFixtureTemplate = `FIXTURE {{.Index}} (
  list position = {{.ListPosition}}
  fixture use type = FIXTURE_USE_INTERIOR
  power adjustment factor = 0.000
  paf desc = None
  lamp wattage = 0.00
  lighting type = LED
  type of fixture = <|{{.Fixture.ID}}|>
  description = <|{{.Fixture.Description}}|>
  fixture type = <|{{.Fixture.ID}}|>
  parent number = 1
  lamp ballast description = <||>
  lamp type = Other
  ballast = UNSPECIFIED_BALLAST
  number of lamps = 1
  fixture wattage = {{.Fixture.VA}}
  quantity = {{.Fixture.Count}} )`
)

func main() {
	inputFile := flag.String("input", "input.csv", "Input file")
	header := flag.Bool("header", true, "first row of csv is header")
	outputFile := flag.String("output", "output.txt", "Output file")

	flag.Parse()

	fixtures := make(map[string]*FixtureInfo)

	// input file is a csv
	file, err := os.Open(*inputFile)
	if err != nil {
		log.Fatal("Error opening file: ", err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 4
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	line := 1
	if *header {
		_, err := reader.Read()
		if err != nil {
			log.Fatal("Error reading header: ", err)
		}
		line++
	}

	// call reader.Read() until EOF
	for record, err := reader.Read(); !errors.Is(err, io.EOF); record, err = reader.Read() {
		if err != nil {
			log.Fatal("Error reading record: ", record, err)
		}
		fixtureId := record[1]
		if _, ok := fixtures[fixtureId]; ok {
			fixtures[fixtureId].Add(1)
		} else {
			// match the digits from record[2] and convert to int
			digits := digitsRegex.FindString(record[3])
			if digits == "" {
				log.Fatalf("Error matching digits in VA: %+v line %d", record, line)
			}

			va, err := strconv.Atoi(digits)
			if err != nil {
				log.Fatalf("Line %d: Error converting VA to int: %v", line, err)
			}

			fixtures[fixtureId] = NewFixtureInfo(fixtureId, record[2], va)
		}
	}
	index := 1
	ListPosition := 1
	// open output file for appending
	outFile, err := os.OpenFile(*outputFile, os.O_APPEND, 0666)

	if err != nil {
		log.Fatal("Error opening output file: ", err)
	}
	defer outFile.Close()
	// apply CCKTemplate to fixtures and write the output to output file
	tmpl, err := template.New("CCKFixtureTemplate").Parse(CCKFixtureTemplate)
	if err != nil {
		log.Fatal("Error parsing template: ", err)
	}
	for _, fixture := range fixtures {
		outFile.WriteString("\r\n")
		data := struct {
			Index        int
			ListPosition int
			Fixture      FixtureInfo
		}{
			Index:        index,
			ListPosition: ListPosition,
			Fixture:      *fixture,
		}
		err := tmpl.Execute(outFile, data)
		if err != nil {
			log.Fatal("Error executing template: ", err)
		}
		index++
		ListPosition++
	}
	fmt.Printf("Wrote %d fixtures to %s\n", len(fixtures), *outputFile)
}
