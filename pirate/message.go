package pirate

type Message struct {
	Header  map[string][]byte
	Metrics []*Metric
}

type Metric struct {
	Name      []byte
	Value     []byte
	Timestamp []byte
}
