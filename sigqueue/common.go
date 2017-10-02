package sigqueue

import "fmt"

type ErrOutOfOrderWait struct {
	id int
}

func NewErrOutOfOrderWait(id int) *ErrOutOfOrderWait {
	return &ErrOutOfOrderWait{id}
}

func (e *ErrOutOfOrderWait) Error() string {
	return fmt.Sprintf("Cannot queue item; ID %d is not increasing", e.id)
}
