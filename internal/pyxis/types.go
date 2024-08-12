package pyxis

type OperatorIndex struct {
	OCPVersion   string `json:"ocp_version"`
	Organization string `json:"organization"`
	EndOfLife    string `json:"end_of_life,omitempty"`
}
