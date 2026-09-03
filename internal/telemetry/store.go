package telemetry

import "context"

type Store interface {
	Append(ctx context.Context, record Record) error
	List(ctx context.Context) ([]Record, error)
}
