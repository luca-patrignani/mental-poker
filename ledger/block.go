package ledger

// Block rappresenta un blocco nella blockchain
type Block struct {
	Index     int         `json:"index"`
	Timestamp int64       `json:"timestamp"`
	PrevHash  string      `json:"prev_hash"`
	Hash      string      `json:"hash"`
	Action    interface{} `json:"action"` // Generic action data
	Votes     []Vote      `json:"votes"`  // Quorum votes
	Metadata  Metadata    `json:"metadata"`
}

type Vote struct {
	VoterID   string `json:"voter_id"`
	Value     string `json:"value"` // "ACCEPT" or "REJECT"
	Signature []byte `json:"signature"`
}

type Metadata struct {
	ProposerID int            `json:"proposer_id"`
	Quorum     int               `json:"quorum"`
	Extra      map[string]string `json:"extra,omitempty"`
}
