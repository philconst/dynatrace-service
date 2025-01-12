package dynatrace

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

const autoTagsPath = "/api/config/v1/autoTags"

type DTTaggingRule struct {
	Name  string  `json:"name"`
	Rules []Rules `json:"rules"`
}
type DynamicKey struct {
	Source string `json:"source"`
	Key    string `json:"key"`
}
type Key struct {
	Attribute  string     `json:"attribute"`
	DynamicKey DynamicKey `json:"dynamicKey"`
	Type       string     `json:"type"`
}
type ComparisonInfo struct {
	Type          string      `json:"type"`
	Operator      string      `json:"operator"`
	Value         interface{} `json:"value"`
	Negate        bool        `json:"negate"`
	CaseSensitive interface{} `json:"caseSensitive"`
}
type Conditions struct {
	Key            Key            `json:"key"`
	ComparisonInfo ComparisonInfo `json:"comparisonInfo"`
}
type Rules struct {
	Type             string       `json:"type"`
	Enabled          bool         `json:"enabled"`
	ValueFormat      string       `json:"valueFormat"`
	PropagationTypes []string     `json:"propagationTypes"`
	Conditions       []Conditions `json:"conditions"`
}

type TagNames struct {
	*StringSet
}

type AutoTagsClient struct {
	client ClientInterface
}

func NewAutoTagClient(client ClientInterface) *AutoTagsClient {
	return &AutoTagsClient{
		client: client,
	}
}

func (atc *AutoTagsClient) Create(rule *DTTaggingRule) error {
	log.WithField("name", rule.Name).Info("Creating DT tagging rule")
	payload, err := json.Marshal(rule)
	if err != nil {
		return err
	}

	_, err = atc.client.Post(autoTagsPath, payload)
	return err
}

func (atc *AutoTagsClient) GetAllTagNames() (*TagNames, error) {
	response, err := atc.client.Get(autoTagsPath)
	if err != nil {
		log.WithError(err).Error("Could not get existing tagging rules")
		return nil, err
	}

	existingDTRules := &listResponse{}
	err = json.Unmarshal(response, existingDTRules)
	if err != nil {
		log.WithError(err).Error("Failed to unmarshal Dynatrace tagging rules")
		return nil, err
	}

	return &TagNames{
		existingDTRules.ToStringSetWith(
			func(values values) string { return values.Name }),
	}, nil
}
