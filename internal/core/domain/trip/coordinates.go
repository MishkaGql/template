package trip

import (
	"math"

	"github.com/MishkaGql/template/internal/core/sharedkernel/errs"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

func NewCoordinates(latitude, longitude float64) (Coordinates, error) {
	coordinates := Coordinates{
		Latitude:  latitude,
		Longitude: longitude,
	}
	if err := coordinates.validate(); err != nil {
		return Coordinates{}, err
	}

	return coordinates, nil
}

func (c Coordinates) validate() error {
	if math.IsNaN(c.Latitude) || math.IsNaN(c.Longitude) ||
		c.Latitude < -90 || c.Latitude > 90 ||
		c.Longitude < -180 || c.Longitude > 180 {
		return errs.NewValidation()
	}

	return nil
}
