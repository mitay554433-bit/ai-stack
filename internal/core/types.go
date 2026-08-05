package core

import "time"

const (
	StateAtGOV    = "G"
	StateApproved = "A"
	StateHeld     = "H"
	StateRejected = "X"
	StateReturned = "B"
	StateAccepted = "F"
)

type Memory struct {
	SourceHash string `json:"h"`
	Codec      string `json:"z"`
	Bytes      int64  `json:"b"`
	Stored     int64  `json:"q"`
	Summary    string `json:"s,omitempty"`
	Provenance string `json:"p,omitempty"`
}

type Validation struct {
	Facts       []string `json:"f,omitempty"`
	Gaps        []string `json:"g,omitempty"`
	Risk        string   `json:"r,omitempty"`
	Recoil      bool     `json:"c"`
	WVC         bool     `json:"w"`
	Reasoner    string   `json:"a,omitempty"`
	ReasonerVer string   `json:"v,omitempty"`
}

type Evolution struct {
	Version    int      `json:"v"`
	Supersedes string   `json:"s,omitempty"`
	Delta      []string `json:"d,omitempty"`
}

type EmergION struct {
	IDN string            `json:"i"`
	STA string            `json:"s"`
	MEM Memory            `json:"m"`
	REL map[string]string `json:"r,omitempty"`
	CAP []string          `json:"c,omitempty"`
	VAL Validation        `json:"v"`
	EVO Evolution         `json:"e"`
}

type DecisionReceipt struct {
	EmergIONID string    `json:"i"`
	Decision   string    `json:"d"`
	Authority  string    `json:"a"`
	Reason     string    `json:"r,omitempty"`
	At         time.Time `json:"t"`
}

type REGReceipt struct {
	EmergIONID string    `json:"i"`
	DecisionID string    `json:"d"`
	At         time.Time `json:"t"`
}

type Event struct {
	Type     string           `json:"t"`
	ID       string           `json:"i"`
	At       time.Time        `json:"a"`
	EmergION *EmergION        `json:"e,omitempty"`
	Decision *DecisionReceipt `json:"d,omitempty"`
	REG      *REGReceipt      `json:"r,omitempty"`
	PrevHash string           `json:"p,omitempty"`
	SelfHash string           `json:"h"`
}

type State struct {
	AtGOV    map[string]EmergION
	Approved map[string]EmergION
	Accepted map[string]EmergION
	Held     map[string]EmergION
	Rejected map[string]EmergION
	Returned map[string]EmergION
	Events   int
	TipHash  string
}

func EmptyState() State {
	return State{
		AtGOV:    map[string]EmergION{},
		Approved: map[string]EmergION{},
		Accepted: map[string]EmergION{},
		Held:     map[string]EmergION{},
		Rejected: map[string]EmergION{},
		Returned: map[string]EmergION{},
	}
}
