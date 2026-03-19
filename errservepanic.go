package webserv

import "fmt"

type errServePanic struct {
	recovered any
	err       error
}

// ErrServePanic matches errors returned by ServeWith when srv.Serve panics.
var ErrServePanic = errServePanic{}

func (e errServePanic) Error() string {
	return fmt.Sprintf("ServeWith(): panic in http.Server.Serve: %v", e.recovered)
}

func (e errServePanic) Is(other error) (yes bool) {
	_, yes = other.(errServePanic)
	return
}

func (e errServePanic) Unwrap() error {
	return e.err
}

func newErrServePanic(recovered any) error {
	var err error
	if recoveredError, ok := recovered.(error); ok {
		err = recoveredError
	}
	return errServePanic{
		recovered: recovered,
		err:       err,
	}
}
