package haws

type State struct {
	State      string                 `json:"state"`
	EntityID   string                 `json:"entity_id"`
	Attributes map[string]interface{} `json:"attributes"`
}

func (c *Client) GetStates() ([]State, error) {
	stateResp := make([]State, 0)

	err := c.sendAndWait(&wsCmd{
		Type: "get_states",
	}, &stateResp)

	return stateResp, err
}
