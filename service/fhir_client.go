package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gopkg.in/mgo.v2"

	fhir "github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/service"
	"gitlab.mitre.org/intervention-engine/redcap-riskservice/models"
)

// PostRiskAssessments posts the risk assessments from the studies to the FHIR server and also stores the risk pies
// to the local Mongo database
func PostRiskAssessments(fhirEndpoint string, studies models.StudyMap, pieCollection *mgo.Collection, basisPieURL string) map[string]error {
	errMap := make(map[string]error)
	for _, study := range studies {
		// Query the FHIR server to find the patient ID by MRN
		r, err := http.NewRequest("GET", fhirEndpoint+"/Patient?identifier="+study.MedicalRecordNumber, nil)
		if err != nil {
			errMap[study.ID] = fmt.Errorf("Couldn't create HTTP request for querying patient with Study ID: %s.  Error: %s", study.ID, err.Error())
			continue
		}
		r.Header.Set("Accept", "application/json")
		res, err := http.DefaultClient.Do(r)
		if err != nil {
			errMap[study.ID] = fmt.Errorf("Couldn't query FHIR server for patient with Study ID: %s.  Error: %s", study.ID, err.Error())
			continue
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			errMap[study.ID] = fmt.Errorf("Received HTTP %d %s from FHIR server when querying patient with Study ID: %s.", res.StatusCode, res.Status, study.ID)
			continue
		}
		var patients fhir.Bundle
		decoder := json.NewDecoder(res.Body)
		if err := decoder.Decode(&patients); err != nil {
			errMap[study.ID] = fmt.Errorf("Couldn't properly decode results from patient query with Study ID: %s.  Error: %s", study.ID, err.Error())
			continue
		}
		if len(patients.Entry) == 0 {
			errMap[study.ID] = fmt.Errorf("Couldn't find patient with MRN %s for Study ID %s", study.MedicalRecordNumber, study.ID)
			continue
		} else if len(patients.Entry) > 1 {
			errMap[study.ID] = fmt.Errorf("Found too many patients (%d) with MRN %s for Study ID %s", len(patients.Entry), study.MedicalRecordNumber, study.ID)
			continue
		}
		patientID := patients.Entry[0].Resource.(*fhir.Patient).Id

		// Get the risk assessments from the records, post to FHIR server, and update pies in Mongo
		results := study.ToRiskServiceCalculationResults(fhirEndpoint + "/Patient/" + patientID)
		service.UpdateRiskAssessmentsAndPies(fhirEndpoint, patientID, results, pieCollection, basisPieURL, REDCapRiskServiceConfig)
	}
	return errMap
}
