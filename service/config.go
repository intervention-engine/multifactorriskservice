package service

import (
	"github.com/intervention-engine/fhir/models"
	"github.com/intervention-engine/riskservice/plugin"
)

var REDCapRiskServiceConfig = plugin.RiskServicePluginConfig{
	Name: "REDCap Risk Service",
	Method: models.CodeableConcept{
		Coding: []models.Coding{{System: "http://interventionengine.org/risk-assessments", Code: "REDCap"}},
		Text:   "REDCap Risk Service",
	},
	PredictedOutcome: models.CodeableConcept{Text: "Unexpected ED/Hospital Visit"},
	DefaultPieSlices: []plugin.Slice{
		{Name: "Clinical Risk", Weight: 25, MaxValue: 4},
		{Name: "Functional and Environmental Risk", Weight: 25, MaxValue: 4},
		{Name: "Psychosocial and Mental Health Risk", Weight: 25, MaxValue: 4},
		{Name: "Utilization Risk", Weight: 25, MaxValue: 4},
	},
	RequiredResourceTypes: []string{},
}
