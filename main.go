package main

import (
	"flag"

	"github.com/davecgh/go-spew/spew"

	"fmt"

	"os"

	"gitlab.mitre.org/intervention-engine/redcap-riskservice/service"
)

func main() {
	rcEndpoint := flag.String("redcapURL", "", "The REDCap endpoint to query for data (required)")
	rcToken := flag.String("redcapToken", "", "The API token to send to the REDCap endpoint (required)")
	flag.Parse()

	if *rcEndpoint == "" || *rcToken == "" {
		flag.PrintDefaults()
	}

	studyMap, err := service.GetREDCapData(*rcEndpoint, *rcToken)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting REDCap data:", err.Error())
		os.Exit(1)
	}

	spew.Dump(studyMap)
}
