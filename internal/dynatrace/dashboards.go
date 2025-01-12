package dynatrace

import (
	log "github.com/sirupsen/logrus"
	"strings"
)

// Dashboards is the data structure for /dashboards endpoint
type Dashboards struct {
	Dashboards []DashboardEntry `json:"dashboards"`
}

// DashboardEntry is the data structure for /dashboards endpoint
type DashboardEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Owner string `json:"owner"`
}

// SearchForDashboardMatching searches for a dashboard that exactly matches project, service and stage
// 	KQG;project=%project%;service=%service%;stage=%stage%;xxx
// It returns the id of the dashboard on success or an empty string otherwise
func (dashboards *Dashboards) SearchForDashboardMatching(project string, stage string, service string) string {
	keyValuePairs := []string{
		strings.ToLower("project=" + project),
		strings.ToLower("stage=" + stage),
		strings.ToLower("service=" + service),
	}

	for _, dashboard := range dashboards.Dashboards {
		// lets see if the dashboard matches our name
		if strings.HasPrefix(strings.ToLower(dashboard.Name), "kqg;") {
			nameSplits := strings.Split(dashboard.Name, ";")

			// now lets see if we can find all our name/value pairs for project, service & stage
			dashboardMatch := true
			for _, findValue := range keyValuePairs {
				foundValue := false
				for _, nameSplitValue := range nameSplits {
					if strings.Compare(findValue, strings.ToLower(nameSplitValue)) == 0 {
						foundValue = true
					}
				}
				if foundValue == false {
					dashboardMatch = false
					continue
				}
			}

			if dashboardMatch {
				return dashboard.ID
			}
		}
	}

	log.WithFields(
		log.Fields{
			"project":        project,
			"stage":          stage,
			"service":        service,
			"dashboardCount": len(dashboards.Dashboards),
		}).Warn("Found dashboards but none matched the name specification")

	return ""
}
