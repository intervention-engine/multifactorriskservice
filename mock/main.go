package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	fhir "github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/service"
	"gitlab.mitre.org/intervention-engine/redcap-riskservice/client"
	"gitlab.mitre.org/intervention-engine/redcap-riskservice/models"
	"gitlab.mitre.org/intervention-engine/redcap-riskservice/server"
	"gopkg.in/mgo.v2"
)

func main() {
	httpAddr := flag.String("http", ":9000", "HTTP service address to listen on")
	mongoAddr := flag.String("mongo", "", "MongoDB address (default: \"mongodb://localhost:27017\")")
	fhirAddr := flag.String("fhir", "", "FHIR API address (required, example: \"http://fhirsrv:3001\")")
	gen := flag.Bool("gen", false, "Flag to indicate that mock risk assessments should be generated immediately")
	flag.Parse()

	if *fhirAddr == "" {
		flag.PrintDefaults()
	}

	// Prefer mongo arg, falling back to env, falling back to default
	mongo := *mongoAddr
	if mongo == "" {
		mongo := os.Getenv("MONGO_PORT_27017_TCP_ADDR")
		if mongo == "" {
			mongo = "mongodb://localhost:27017"
		}
	} else if strings.HasPrefix(mongo, ":") {
		mongo = "mongodb://localhost" + mongo
	}
	session, err := mgo.Dial(mongo)
	if err != nil {
		panic("Can't connect to the database")
	}
	defer session.Close()
	db := session.DB("mock-riskservice")
	pieCollection := db.C("pies")

	// Get own endpoint address, falling back to discovery if needed
	endpoint := *httpAddr
	if strings.HasPrefix(endpoint, ":") {
		endpoint = discoverSelf()
	}
	basisPieURL := endpoint + "/pies/"

	// Create the gin engine, register the routes, and run!
	e := gin.Default()
	RegisterMockRoutes(e, *fhirAddr, pieCollection, basisPieURL)

	if *gen {
		results, err := RefreshMockRiskAssessments(*fhirAddr, pieCollection, basisPieURL)
		if err != nil {
			log.Println("Failed to generate mock risk assessments", err)
		} else {
			client.LogResultSummary(results)
		}
	}
	e.Run(*httpAddr)
}

// RegisterMockRoutes sets up the http request handlers for the mock service with Gin
func RegisterMockRoutes(e *gin.Engine, fhirEndpoint string, pieCollection *mgo.Collection, basisPieURL string) {
	server.RegisterPieHandler(e, pieCollection)
	RegisterMockRefreshHandler(e, fhirEndpoint, pieCollection, basisPieURL)
}

// RegisterMockRefreshHandler registers the handler to refresh mock risk assessments
func RegisterMockRefreshHandler(e *gin.Engine, fhirEndpoint string, pieCollection *mgo.Collection, basisPieURL string) {
	e.POST("/refresh", func(c *gin.Context) {
		results, err := RefreshMockRiskAssessments(fhirEndpoint, pieCollection, basisPieURL)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		client.LogResultSummary(results)
		c.JSON(http.StatusOK, results)
	})
}

var m sync.Mutex

// RefreshMockRiskAssessments pulls the risk assessment data from REDCap and posts it to the FHIR server, replacing older
// risk assessments and storing pie representations.
func RefreshMockRiskAssessments(fhirEndpoint string, pieCollection *mgo.Collection, basisPieURL string) ([]client.Result, error) {
	m.Lock()
	defer m.Unlock()

	pMap, err := getPatientSummariesFromFHIR(fhirEndpoint)
	if err != nil {
		return nil, err
	}

	results := make([]client.Result, 0, len(pMap))
	for id, sum := range pMap {
		study := sum.ToStudy()
		result := client.Result{
			StudyID:             study.ID,
			MedicalRecordNumber: study.MedicalRecordNumber,
			FHIRPatientID:       id,
		}
		calcResults := study.ToRiskServiceCalculationResults(fhirEndpoint + "/Patient/" + id)
		err = service.UpdateRiskAssessmentsAndPies(fhirEndpoint, id, calcResults, pieCollection, basisPieURL, client.REDCapRiskServiceConfig)
		if err != nil {
			result.Error = err
		} else {
			result.RiskAssessmentCount = len(calcResults)
		}
		results = append(results, result)
	}

	return results, nil
}

func getPatientSummariesFromFHIR(fhirEndpoint string) (map[string]patientSummary, error) {
	pMap := make(map[string]patientSummary)
	query := fhirEndpoint + "/Patient?_revinclude=Condition:patient&_revinclude=MedicationStatement:patient"
	// Perform a loop to go through the pages of a bundle response
	for true {
		// Query the FHIR server to get the patients
		r, err := http.NewRequest("GET", query, nil)
		if err != nil {
			return nil, err
		}
		r.Header.Set("Accept", "application/json")
		res, err := http.DefaultClient.Do(r)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Received HTTP %d %s from FHIR server when querying for patients.", res.StatusCode, res.Status)
		}
		var bundle fhir.Bundle
		decoder := json.NewDecoder(res.Body)
		if err := decoder.Decode(&bundle); err != nil {
			return nil, err
		}
		for _, entry := range bundle.Entry {
			var sum patientSummary
			switch t := entry.Resource.(type) {
			case *fhir.Patient:
				sum = pMap[t.Id]
				sum.ID = t.Id
				if t.BirthDate != nil {
					// Approximate age (not perfect, but good enough)
					sum.Age = int(time.Since(t.BirthDate.Time).Hours() / (24 * 365))
				}
			case *fhir.Condition:
				sum = pMap[t.Patient.ReferencedID]
				sum.ID = t.Patient.ReferencedID
				sum.ConditionCount += sum.ConditionCount
			case *fhir.MedicationStatement:
				sum = pMap[t.Patient.ReferencedID]
				sum.ID = t.Patient.ReferencedID
				sum.MedicationCount += sum.MedicationCount
			}
			if sum.ID != "" {
				pMap[sum.ID] = sum
			}
		}
		var more bool
		for _, link := range bundle.Link {
			if link.Relation == "next" && link.Url != "" {
				query = link.Url
				more = true
			}
		}
		if !more {
			break
		}
	}

	return pMap, nil
}

type patientSummary struct {
	ID              string
	Age             int
	ConditionCount  int
	MedicationCount int
}

func (p *patientSummary) ToStudy() models.Study {
	var study models.Study
	study.ID = p.ID
	study.MedicalRecordNumber = p.ID
	for d := time.Date(2014, time.June, 1, 12, 0, 0, 0, time.Local); d.Before(time.Now()); {
		var record models.Record
		record.StudyID = p.ID
		record.MedicalRecordNumber = p.ID
		record.ParticipantInfoComplete = "2"
		record.RiskFactorDate = d.Format("2006-01-02")
		record.RiskFactorsComplete = "2"
		if len(study.Records) == 0 {
			p.populateInitialRecord(&record)
		} else {
			p.populateNextRecord(&record, study.Records[len(study.Records)-1], study.Records[0])
		}
		study.Records = append(study.Records, record)
		//log.Printf("%s: %s [C: %s, F: %s, P: %s, U: %s]\n", record.RiskFactorDate, record.PerceivedRisk, record.ClinicalRisk, record.FunctionalRisk, record.PsychosocialRisk, record.UtilizationRisk)

		switch record.PerceivedRisk {
		case "1":
			d = d.AddDate(0, 3, 0)
		case "2":
			d = d.AddDate(0, 2, 0)
		case "3":
			d = d.AddDate(0, 0, 21)
		case "4":
			d = d.AddDate(0, 0, 7)
		}
	}
	return study
}

func (p *patientSummary) populateInitialRecord(record *models.Record) {
	total := p.ConditionCount + p.MedicationCount
	switch {
	case total < 3:
		record.ClinicalRisk = "1"
	case total < 6:
		record.ClinicalRisk = "2"
	default:
		record.ClinicalRisk = "3"
	}
	record.FunctionalRisk = randomishScore()
	record.PsychosocialRisk = randomishScore()
	record.UtilizationRisk = randomishScore()
	populatePerceivedRisk(record)
}

func (p *patientSummary) populateNextRecord(record *models.Record, previous models.Record, initial models.Record) {
	// Clinical low / high should be within one point of original score
	cLowInt, _ := strconv.Atoi(initial.ClinicalRisk)
	cHighInt := cLowInt
	if cLowInt != 1 {
		cLowInt--
	}
	if cHighInt != 4 {
		cHighInt++
	}
	record.ClinicalRisk = nextScore(previous.ClinicalRisk, fmt.Sprint(cLowInt), fmt.Sprint(cHighInt))
	record.FunctionalRisk = nextScore(previous.FunctionalRisk, "1", "4")
	record.PsychosocialRisk = nextScore(previous.PsychosocialRisk, "1", "4")
	record.UtilizationRisk = nextScore(previous.UtilizationRisk, "1", "4")
	populatePerceivedRisk(record)
}

func populatePerceivedRisk(record *models.Record) {
	for _, risk := range []string{record.ClinicalRisk, record.FunctionalRisk, record.PsychosocialRisk, record.UtilizationRisk} {
		if risk > record.PerceivedRisk {
			record.PerceivedRisk = risk
		}
	}
}

var r = rand.New(rand.NewSource(time.Now().Unix()))

func randomishScore() string {
	i := r.Intn(100)
	switch {
	case i < 5:
		// 5% chance
		return "4"
	case i < 20:
		// 15% chance
		return "3"
	case i < 50:
		// 30% chance
		return "2"
	}
	// 50% chance
	return "1"
}

func nextScore(previous, low, high string) string {
	next := previous

	i := r.Intn(100)
	switch previous {
	case "1":
		if i < 10 {
			next = "2"
		}
	case "2":
		if i < 30 {
			next = "1"
		} else if i < 50 {
			next = "3"
		}
	case "3":
		if i < 50 {
			next = "2"
		} else if i < 65 {
			next = "4"
		}
	case "4":
		if i < 50 {
			next = "3"
		}
	}
	if next < low || next > high {
		// Try it again
		return nextScore(previous, low, high)
	}
	return next
}

func discoverSelf() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Println("Unable to determine IP address.  Defaulting to localhost.")
		return "http://localhost:9000"
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return "http://" + ipnet.IP.String() + ":9000"
			}
		}
	}

	log.Println("Unable to determine IP address.  Defaulting to localhost.")
	return "http://localhost:9000"
}
