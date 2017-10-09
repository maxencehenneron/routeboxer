# Golang routeboxer

This is a Golang implementation of the RouteBoxer Class from Google.

## Install

```
go get github.com/dernise/routeboxer
```

## Usage

You first need to initialize a RouteBoxer object with an array of coordinates [Longitude, Latitude] representing the route and the desired range.

```go
	pointSet := geo.PointSet{
		{3.0366159982599186, 50.627300916239626}, {3.0368849735327217, 50.626974944025285}
	}

	routeBoxer := NewRouteBoxer(1000, pointSet)
```

You can now run the algorithm with :

```go
	boxes := routeBoxer.Boxes()
```

Once the calculation are complete, you can get a GeoJSON like object

```go
	boxes.ToGeoJson()
```
