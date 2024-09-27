package portForwarder

import (
	"fmt"
	"log"
	"time"

	"astuart.co/edgeos-rest/pkg/edgeos"
	"github.com/mitchellh/mapstructure"
)

type Port struct {
	Port             string
	ForwardToAddress string `mapstructure:"forward_to_ip"`
	ForwardToPort    string `mapstructure:"forward_to_port"`
	Protocol         string
	Description      string
}

func NewPort(port, forwardToAddress, forwardToPort, protocol, description string) *Port {
	return &Port{
		Port:             port,
		ForwardToAddress: forwardToAddress,
		ForwardToPort:    forwardToPort,
		Protocol:         protocol,
		Description:      description,
	}
}

func (p *Port) UnmarshalMap(m map[string]string) error {
	return mapstructure.Decode(m, p)
}

type PortCredentials struct {
	RouterIp string `config:"router_ip"`
	Username string
	Password string
}

type EdgeOsRouterPortForwarder struct {
	EdgeClient *edgeos.Client
	creds      PortCredentials
}

func NewEdgeOsRouterPortForwarder(creds PortCredentials) *EdgeOsRouterPortForwarder {
	return &EdgeOsRouterPortForwarder{
		EdgeClient: nil,
		creds:      creds,
	}
}

func (r *EdgeOsRouterPortForwarder) Connect() {
	client, err := edgeos.NewClient(r.creds.RouterIp, r.creds.Username, r.creds.Password)
	if err != nil {
		log.Fatal(err)
	}

	if err := client.Login(); err != nil {
		log.Fatal(err)
	}

	r.EdgeClient = client
}

func (r *EdgeOsRouterPortForwarder) emptyRule(rule edgeos.Resp) bool {
	return rule["data"].(map[string]interface{})["rules-config"] == nil
}

func (r *EdgeOsRouterPortForwarder) portExists(data []interface{}, item Port) (bool, int) {

	for i, d := range data {
		// Type check to handle both map[string]interface{} and map[string]string
		var dMap map[string]interface{}

		switch v := d.(type) {
		case map[string]interface{}:
			dMap = v
		case map[string]string:
			// Convert map[string]string to map[string]interface{}
			dMap = make(map[string]interface{})
			for key, value := range v {
				dMap[key] = value
			}
		default:
			// If the type is neither, skip or handle error
			continue
		}

		if dMap["description"] == item.Description && dMap["forward-to-port"] == item.ForwardToPort &&
			dMap["forward-to-address"] == item.ForwardToAddress && dMap["protocol"] == item.Protocol &&
			// d["original-port"] == strconv.Itoa(int(item.Port)) {
			dMap["original-port"] == item.Port {
			return true, i
		}
	}
	return false, -1
}

func (r *EdgeOsRouterPortForwarder) GetFeature(feature edgeos.Scenario) edgeos.Resp {
	feat, err := r.EdgeClient.Feature(feature)
	if err != nil {
		log.Fatal(err)
	}

	for r.emptyRule(feat) {
		time.Sleep(2 * time.Second)
		feat, err = r.EdgeClient.Feature(edgeos.PortForwarding)
		if err != nil {
			log.Fatal(err)
		}
	}

	return feat
}

func (r *EdgeOsRouterPortForwarder) GetForwardedPorts() []map[string]string {
	feat := r.GetFeature(edgeos.PortForwarding)

	rawRules := feat["data"].(map[string]interface{})["rules-config"].([]interface{})

	var rules []map[string]string
	for _, rawRule := range rawRules {
		ruleMap, ok := rawRule.(map[string]interface{})
		if !ok {
			continue
		}

		stringRule := make(map[string]string)
		for k, v := range ruleMap {
			if str, ok := v.(string); ok {
				stringRule[k] = str
				fmt.Printf("stringRule[%q] = %q\n", k, str)
			} else {
				fmt.Printf("Warning: Unexpected value type for key %q\n", k)
			}
		}
		rules = append(rules, stringRule)
	}

	return rules
}

func (r *EdgeOsRouterPortForwarder) AddPorts(portsToForward []Port) int {
	feat := r.GetFeature(edgeos.PortForwarding)

	rulesConfig := feat["data"].(map[string]interface{})["rules-config"]

	var d []interface{}
	switch v := rulesConfig.(type) {
	case []interface{}:
		d = v
	case string:
		fmt.Println("Emtpy rules-config")
		d = []interface{}{}
	default:
		log.Fatalf("Unexpected type for rules-config: %T", v)
	}

	portsAdded := 0

	for _, v := range portsToForward {

		portForwards := make(map[string]string)
		portForwards["description"] = v.Description
		portForwards["forward-to-address"] = v.ForwardToAddress
		// portForwards["forward-to-port"] = strconv.Itoa(int(v.ForwardToPort))
		// portForwards["original-port"] = strconv.Itoa(int(v.Port))
		portForwards["forward-to-port"] = v.ForwardToPort
		portForwards["original-port"] = v.Port
		portForwards["protocol"] = v.Protocol

		if exists, _ := r.portExists(d, v); !exists {
			portsAdded++
			d = append(d, portForwards)
		}
	}

	// Update the rules-config feature
	if portsAdded > 0 {
		feat["data"].(map[string]interface{})["rules-config"] = d
		r.EdgeClient.SetFeature(edgeos.PortForwarding, feat["data"])
	}

	return portsAdded
}

func (r *EdgeOsRouterPortForwarder) RemovePorts(portsToRemove []Port) int {

	feat := r.GetFeature(edgeos.PortForwarding)
	removedPorts := 0

	d := feat["data"].(map[string]interface{})["rules-config"].([]interface{})
	for _, rule := range portsToRemove {
		if exists, idx := r.portExists(d, rule); exists {

			d = append(d[:idx], d[idx+1:]...)
			removedPorts++
		}
	}

	if removedPorts > 0 {
		feat["data"].(map[string]interface{})["rules-config"] = d
		r.EdgeClient.SetFeature(edgeos.PortForwarding, feat["data"])
	}

	return removedPorts
}

func (r *EdgeOsRouterPortForwarder) RemoveAllPorts() int {
	feat := r.GetFeature(edgeos.PortForwarding)
	d := feat["data"].(map[string]interface{})["rules-config"].([]interface{})
	removedPorts := len(d)

	feat["data"].(map[string]interface{})["rules-config"] = []interface{}{}
	r.EdgeClient.SetFeature(edgeos.PortForwarding, feat["data"])

	return removedPorts
}
