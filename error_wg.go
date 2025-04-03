package virtualfs

import "sync"

type errorWG struct {
	err chan error
	wg  *sync.WaitGroup
}

func newErrorWG() *errorWG {
	return &errorWG{
		err: make(chan error, 1),
		wg:  new(sync.WaitGroup),
	}
}

func (e *errorWG) wait() error {
	e.wg.Wait()
	select {
	case err := <-e.err:
		return err
	default:
		return nil
	}
}

func (e *errorWG) run(f func() error) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		err := f()
		if err != nil {
			select {
			case e.err <- err:
			default:
				// if already an error just move on so we arent blocking
				// maybe should do array but then needs to be thread safe
			}
		}
	}()
}
