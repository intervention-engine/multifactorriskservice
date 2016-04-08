package models

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/intervention-engine/riskservice/plugin"
)

// Record represents the key info from a REDCap record in the risk stratification project
type Record struct {
	StudyID                 interface{} `json:"study_id"`
	EventName               string      `json:"redcap_event_name"`
	MedicalRecordNumber     string      `json:"mrn"`
	ParticipantInfoComplete string      `json:"participant_information_complete"`

	RiskFactorDate      string `json:"rf_date"`
	ClinicalRisk        string `json:"rf_cmc_risk_cat"`
	FunctionalRisk      string `json:"rf_func_risk_cat"`
	PsychosocialRisk    string `json:"rf_sb_risk_cat"`
	UtilizationRisk     string `json:"rf_util_risk_cat"`
	PerceivedRisk       string `json:"rf_risk_predicted"`
	RiskFactorsComplete string `json:"risk_factors_complete"`
}

// StudyIDString returns a string representation of the study ID (which could be a string or a number)
func (r *Record) StudyIDString() string {
	return fmt.Sprint(r.StudyID)
}

// RiskFactorDateTime returns the parsed date/time for the risk factor form
func (r *Record) RiskFactorDateTime() (time.Time, error) {
	return time.ParseInLocation("2006-01-02", r.RiskFactorDate, time.Local)
}

// IsParticipationInfoComplete checks that the participation info form was marked as complete AND that a medical
// record number exists
func (r *Record) IsParticipationInfoComplete() bool {
	return r.ParticipantInfoComplete == "2" && r.MedicalRecordNumber != ""
}

// IsRiskFactorsComplete checks that the risk factors form was marked as complete, that a valid risk factor date was
// set, and that all risk factor scores are set
func (r *Record) IsRiskFactorsComplete() bool {
	return r.RiskFactorsComplete == "2" && r.RiskFactorDate != "" && r.ClinicalRisk != "" && r.FunctionalRisk != "" &&
		r.PsychosocialRisk != "" && r.UtilizationRisk != "" && r.PerceivedRisk != ""
}

// ToPie converts the record to the Intervention Engine pie format used for identifying risk components
func (r *Record) ToPie() (pie *plugin.Pie, err error) {
	if !r.IsRiskFactorsComplete() {
		return nil, errors.New("Cannot create a pie with incomplete risk factors")
	}

	pie = new(plugin.Pie)
	pie.Id = bson.NewObjectId()
	pie.Created = time.Now()

	crSlice, err := newSlice("Clinical Risk", r.ClinicalRisk)
	if err != nil {
		return nil, err
	}

	frSlice, err := newSlice("Functional and Environmental Risk", r.FunctionalRisk)
	if err != nil {
		return nil, err
	}

	prSlice, err := newSlice("Psychosocial and Mental Health Risk", r.PsychosocialRisk)
	if err != nil {
		return nil, err
	}

	urSlice, err := newSlice("Utilization Risk", r.UtilizationRisk)
	if err != nil {
		return nil, err
	}

	pie.Slices = []plugin.Slice{*crSlice, *frSlice, *prSlice, *urSlice}

	return pie, nil
}

func newSlice(name string, score string) (slice *plugin.Slice, err error) {
	value, err := strconv.Atoi(score)
	if err != nil {
		return nil, fmt.Errorf("Invalid %s: %s", name, score)
	}
	slice = new(plugin.Slice)
	slice.Name = name
	slice.Value = value
	slice.Weight = 25
	slice.MaxValue = 4

	return
}
