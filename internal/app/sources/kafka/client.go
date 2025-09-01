package kafka

type Kafka struct {
	name string
}

func (q *Kafka) Fetch() {}

func (q *Kafka) Name() string {
	return q.name
}
