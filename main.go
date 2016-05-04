package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron"

	"gopkg.in/mgo.v2"

	"github.com/intervention-engine/multifactorriskservice/server"
)

func main() {
	httpAddr := flag.String("http", ":9000", "HTTP service address to listen on")
	mongoAddr := flag.String("mongo", "", "MongoDB address (default: \"mongodb://localhost:27017\")")
	fhirAddr := flag.String("fhir", "", "FHIR API address (required, example: \"http://fhirsrv:3001\")")
	redcapAddr := flag.String("redcap", "", "REDCap API address (required, example: \"http://redcapsrv:80\")")
	token := flag.String("token", "", "REDCap API token (required)")
	cronSpec := flag.String("cron", "0 0 22 * * *", "Cron expression indicating when risk assessments should be automatically refreshed")
	flag.Parse()

	if *fhirAddr == "" || *redcapAddr == "" || *token == "" {
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
	db := session.DB("riskservice")
	pieCollection := db.C("pies")

	// Get own endpoint address, falling back to discovery if needed
	endpoint := *httpAddr
	if strings.HasPrefix(endpoint, ":") {
		endpoint = discoverSelf()
	}
	basisPieURL := endpoint + "/pies/"

	// Setup the cron job and start the scheduler
	c := cron.New()
	err = server.ScheduleRefreshRiskAssessmentsCron(c, *cronSpec, *fhirAddr, *redcapAddr, *token, pieCollection, basisPieURL)
	if err != nil {
		panic("Can't setup cron job for refreshing risk assessments.  Specified spec: " + *cronSpec)
	}
	c.Start()
	defer c.Stop()

	// Create the gin engine, register the routes, and run!
	e := gin.Default()
	server.RegisterRoutes(e, *fhirAddr, *redcapAddr, *token, pieCollection, basisPieURL)
	e.Run(*httpAddr)
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
