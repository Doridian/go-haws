package haws

type CallServiceTarget struct {
	DeviceID []string `json:"device_id,omitempty"`
	EntityID []string `json:"entity_id,omitempty"`
	AreaID   []string `json:"area_id,omitempty"`
}

type wsComandCallService struct {
	wsCmd

	Domain      string                 `json:"domain"`
	Service     string                 `json:"service"`
	ServiceData map[string]interface{} `json:"service_data,omitempty"`
	Target      *CallServiceTarget     `json:"target,omitempty"`
}

func (c *Client) CallService(domain string, service string, serviceData map[string]interface{}, target *CallServiceTarget) error {
	return c.sendAndWait(&wsComandCallService{
		wsCmd: wsCmd{
			Type: "call_service",
		},
		Domain:      domain,
		Service:     service,
		ServiceData: serviceData,
		Target:      target,
	}, nil)
}
