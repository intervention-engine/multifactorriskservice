package service

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"gitlab.mitre.org/intervention-engine/redcap-riskservice/models"
)

// GetREDCapData queries REDCap at the specified endpoint with the specifed token, returning a StudyMap containing
// the resulting data.
func GetREDCapData(endpoint string, token string) (models.StudyMap, error) {
	form := url.Values{}
	form.Set("token", token)
	form.Set("content", "record")
	form.Set("format", "json")
	form.Set("returnFormat", "json")
	form.Set("type", "flat")
	form.Set("fields", "study_id, redcap_event_name, mrn, participant_information_complete, rf_date, rf_cmc_risk_cat, rf_func_risk_cat, rf_sb_risk_cat, rf_util_risk_cat, rf_risk_predicted, risk_factors_complete")

	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	res, err := http.DefaultClient.PostForm(endpoint, form)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var records []models.Record
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&records); err != nil {
		return nil, err
	}

	m := make(models.StudyMap)
	if err := m.AddRecords(records); err != nil {
		return nil, err
	}

	return m, nil
}
