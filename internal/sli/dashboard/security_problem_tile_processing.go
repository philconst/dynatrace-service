package dashboard

import (
	"fmt"
	"github.com/keptn-contrib/dynatrace-service/internal/common"
	"github.com/keptn-contrib/dynatrace-service/internal/dynatrace"
	keptnv2 "github.com/keptn/go-utils/pkg/lib/v0_2_0"
	log "github.com/sirupsen/logrus"
	"time"
)

type SecurityProblemTileProcessing struct {
	client    dynatrace.ClientInterface
	startUnix time.Time
	endUnix   time.Time
}

func NewSecurityProblemTileProcessing(client dynatrace.ClientInterface, startUnix time.Time, endUnix time.Time) *SecurityProblemTileProcessing {
	return &SecurityProblemTileProcessing{
		client:    client,
		startUnix: startUnix,
		endUnix:   endUnix,
	}
}

func (p *SecurityProblemTileProcessing) Process(tile *dynatrace.Tile, dashboardFilter *dynatrace.DashboardFilter) *TileResult {

	// get the tile specific management zone filter that might be needed by different tile processors
	// Check for tile management zone filter - this would overwrite the dashboardManagementZoneFilter
	tileManagementZoneFilter := NewManagementZoneFilter(dashboardFilter, tile.TileFilter.ManagementZone)

	// we will query the number of open security problems based on the specification of that tile
	problemSelector := "status(OPEN)" + tileManagementZoneFilter.ForProblemSelector()

	tileResult, err := p.processProblemSelector(problemSelector, p.startUnix, p.endUnix)
	if err != nil {
		log.WithError(err).Error("Error Processing OPEN_SECURITY_PROBLEMS")
		return nil
	}

	return tileResult
}

// processProblemSelector Processes an Open Problem Tile and queries the number of open problems. The current default is that there is a pass criteria of <= 0 as we dont allow problems
// If successful returns sliResult, sliIndicatorName, sliQuery & sloDefinition
func (p *SecurityProblemTileProcessing) processProblemSelector(securityProblemSelector string, startUnix time.Time, endUnix time.Time) (*TileResult, error) {

	problemQuery := ""
	if securityProblemSelector != "" {
		problemQuery = fmt.Sprintf("securityProblemSelector=%s", securityProblemSelector)
	}

	// Step 1: Query the Dynatrace API to get the number of actual problems matching that query and timeframe
	problemQueryResult, err := dynatrace.NewSecurityProblemsClient(p.client).GetByQuery(problemQuery, startUnix, endUnix)
	if err != nil {
		return nil, err
	}

	// Step 2: As we have the SLO Result including SLO Definition we add it to the SLI & SLO objects
	// IndicatorName is based on the slo Name
	// the value defaults to the E
	indicatorName := "security_problems"
	value := float64(problemQueryResult.TotalCount)
	sliResult := &keptnv2.SLIResult{
		Metric:  indicatorName,
		Value:   value,
		Success: true,
	}

	log.WithFields(
		log.Fields{
			"indicatorName": indicatorName,
			"value":         value,
		}).Debug("Adding SLO to sloResult")

	// add this to our SLI Indicator JSON in case we need to generate an SLI.yaml
	// we prepend this with SECPV2;entitySelector=asdaf&problemSelector=asdf
	sliQuery := fmt.Sprintf("SECPV2;%s", problemQuery)

	// lets add the SLO definitin in case we need to generate an SLO.yaml
	// we normally parse these values from the tile name. In this case we just build that tile name -> maybe in the future we will allow users to add additional SLO defs via the Tile Name, e.g: weight or KeySli
	sloString := fmt.Sprintf("sli=%s;pass=<=0;key=true", indicatorName)
	sloDefinition := common.ParsePassAndWarningWithoutDefaultsFrom(sloString)

	return &TileResult{
		sliResult: sliResult,
		objective: sloDefinition,
		sliName:   indicatorName,
		sliQuery:  sliQuery,
	}, nil
}
