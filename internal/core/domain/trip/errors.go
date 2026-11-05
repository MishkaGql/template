package trip

import "github.com/MishkaGql/template/internal/core/sharedkernel/errs"

const (
	CodeDriverBusy           errs.ErrorCode = "driver_busy"
	CodeTripNotFound         errs.ErrorCode = "trip_not_found"
	CodeTripAlreadyCompleted errs.ErrorCode = "trip_completed"
	CodeIdempotencyConflict  errs.ErrorCode = "idempotency_conflict"
)
