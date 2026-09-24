package projection

import (
	"context"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/eventsource/subscription"
	"github.com/kduong-dev/goutil/fatal"
)

type Projection struct {
	log    eventsource.Log
	apply  func(ctx context.Context, event *eventsource.Event) error
	cursor int64
}

type NewInput struct {
	Log   eventsource.Log
	Apply func(ctx context.Context, event *eventsource.Event) error
}

func New(input NewInput) *Projection {
	return &Projection{
		log:   input.Log,
		apply: input.Apply,
	}
}

func (projection *Projection) CatchUp(ctx context.Context) {
	var err error
	projection.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    projection.log,
		Cursor: projection.cursor,
		Apply:  projection.apply,
	})
	fatal.OnError(err)
}

func (projection *Projection) Append(ctx context.Context, frame any) (err error) {
	data := fatal.UnlessMarshal(frame)
	_, err = projection.log.Append(data)
	return
}
