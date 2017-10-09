package routeboxer

import (
	"math"

	"github.com/paulmach/go.geo"
)

func RhumbDestinationPoint(currentPoint geo.Point, brng float64, dist float64) *geo.Point {
	d := dist / geo.EarthRadius
	lat1, lon1 := deg2rad(currentPoint.Lat()), deg2rad(currentPoint.Lng())
	brng = deg2rad(brng)

	dLat := d * math.Cos(brng)

	if math.Abs(dLat) < 1e-10 {
		dLat = 0 // dLat < 1 mm
	}

	lat2 := lat1 + dLat
	dPhi := math.Log(math.Tan(lat2/2+math.Pi/4) / math.Tan(lat1/2+math.Pi/4))

	var q float64
	if dPhi != 0 {
		q = dLat / dPhi
	} else {
		q = math.Cos(lat1)
	}
	dLon := d * math.Sin(brng) / q

	if math.Abs(lat2) > math.Pi/2 {
		if lat2 > 0 {
			lat2 = math.Pi - lat2
		} else {
			lat2 = -math.Pi - lat2
		}
	}

	lon2 := math.Remainder(lon1+dLon+3*math.Pi, (2*math.Pi)-math.Pi)

	return geo.NewPointFromLatLng(rad2deg(lat2), rad2deg(lon2))
}

func RhumBearingTo(point geo.Point, dest geo.Point) float64 {
	var dLon = deg2rad(dest.Lng() - point.Lng())
	var dPhi = math.Log(math.Tan(deg2rad(dest.Lat())/2+math.Pi/4) / math.Tan(deg2rad(point.Lat())/2+math.Pi/4))
	if math.Abs(dLon) > math.Pi {
		if dLon > 0 {
			dLon = -(2*math.Pi - dLon)
		} else {
			dLon = 2*math.Pi + dLon
		}
	}

	return math.Remainder(rad2deg(math.Atan2(dLon, dPhi))+360, 360)
}

func deg2rad(d float64) float64 {
	return d * math.Pi / 180.0
}

func rad2deg(r float64) float64 {
	return 180.0 * r / math.Pi
}
