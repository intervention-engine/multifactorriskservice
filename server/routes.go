package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/intervention-engine/riskservice/plugin"
	"gitlab.mitre.org/intervention-engine/redcap-riskservice/client"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// RegisterRoutes sets up the http request handlers with Gin
func RegisterRoutes(e *gin.Engine, db *mgo.Database, endpoint, fhirEndpoint, redcapEndpoint, redcapToken string) {
	pieCollection := db.C("pies")

	e.GET("/pies/:id", func(c *gin.Context) {
		pie := &plugin.Pie{}
		id := c.Param("id")
		if bson.IsObjectIdHex(id) {
			query := pieCollection.FindId(bson.ObjectIdHex(id))
			if err := query.One(pie); err == nil {
				c.JSON(http.StatusOK, pie)
			} else {
				c.Status(http.StatusNotFound)
			}
		} else {
			c.String(http.StatusBadRequest, "Bad ID format for requested Pie. Should be a BSON Id")
		}
		return
	})

	e.POST("/refresh", func(c *gin.Context) {
		results, err := client.RefreshRiskAssessments(fhirEndpoint, redcapEndpoint, redcapToken, pieCollection, endpoint+"/pies/")
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, results)
	})
}
