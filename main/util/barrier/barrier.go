package barrier

type Barrier struct {
	queue chan interface{}
}

// TODO wider capacity type
func New(capacity int) Barrier {
	return Barrier{
		queue: make(chan interface{}, capacity),
	}
}

func (this *Barrier) CheckIn() {
	this.queue <- nil
}

func (this *Barrier) Await(onCheckIn ...func(int)) {
	for i := 0; i < cap(this.queue); i++ {
		<-this.queue
		for _, cb := range onCheckIn {
			cb(i)
		}
	}
}