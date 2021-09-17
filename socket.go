package pulp

import (
	"context"
	"fmt"
	"sync"
)

type Socket struct {
	ID uint32

	updates   chan socketUpdate
	lastState LiveComponent
	Err       error
	context.Context
	events chan<- Event

	once sync.Once

	// assets struct {
	// 	// currentRoute string
	// 	flash struct {
	// 		err, warning, info *string
	// 	}
	// }

	Route string
}

type socketUpdate struct {
	ctx       context.Context
	ch        chan<- socketUpdate
	component *LiveComponent
	route     *string // eh. not sure if this will stay
}

type Assets map[string]interface{}

func (a Assets) mergeAndOverwrite(other Assets) Assets {
	for key, val := range other {
		a[key] = val
	}
	return a
}

type M map[string]interface{}

// don't use this yet. this is not working perfectly
func (s *Socket) Dispatch(event string, data M) {
	select {
	case <-s.Done():
	case s.events <- UserEvent{Name: event, Data: data}:
	}
}

func (s *Socket) Errorf(format string, values ...interface{}) *Socket {
	s.Err = fmt.Errorf(format, values...)
	return s
}

// func (s *Socket) Changes(state LiveComponent) *Socket {
// 	s.lastState = state
// 	return s
// }

func (s *Socket) FlashError(route string) {
}

func (s *Socket) FlashInfo(route string) {
}

func (s *Socket) FlashWarning(route string) {
}

// func (s *Socket) Redirect(route string) *Socket {
// 	s.Route = route
// 	return s
// }

func (s Socket) assets() Assets {
	return Assets{
		"route": s.Route,
	}
}

func (s *Socket) Prepare() *socketUpdate {
	return &socketUpdate{ch: s.updates, ctx: s.Context}
}

func (u socketUpdate) apply(socket *Socket) {
	if u.route != nil {
		socket.Route = *u.route
	}

	if u.component != nil {
		socket.lastState = *u.component
	}
}

func (u *socketUpdate) Redirect(route string) *socketUpdate {
	u.route = &route
	return u
}

func (u *socketUpdate) Changes(c LiveComponent) *socketUpdate {
	u.component = &c
	return u
}

func (s *socketUpdate) Do() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("socket panic: \n")
			}
		}()

		select {
		case <-s.ctx.Done():
		case s.ch <- *s:
		}
	}()
}
