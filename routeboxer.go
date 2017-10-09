package routeboxer

import (
	"math"

	"github.com/paulmach/go.geo"
)

type RouteBoxer struct {
	distanceRange float64      // The distance in kms around the route that the generated boxes must cover.
	vertices      geo.PointSet // Array of LatLngs representing the vertices of the path

	latGrid []float64 // Array that holds the latitude coordinate of each vertical grid line
	lngGrid []float64 // Array that holds the longitude coordinate of each horizontal grid line
	grid    [][]int
	boxesX  []geo.Bound
	boxesY  []geo.Bound
}

type RouteBoxerResult []geo.Bound

type GeoJsonResult struct {
	Type        string          `json:"type"`
	Coordinates [][][]geo.Point `json:"coordinates"`
}

func NewRouteBoxer(distanceRange float64, vertices geo.PointSet) *RouteBoxer {
	return &RouteBoxer{distanceRange, vertices, []float64{}, []float64{}, nil, []geo.Bound{}, []geo.Bound{}}
}

func (r *RouteBoxer) Boxes() RouteBoxerResult {
	r.buildGrid()

	r.FindIntersectingCells()

	r.mergeIntersectingCells()

	if len(r.boxesX) <= len(r.boxesY) {
		return r.boxesX
	} else {
		return r.boxesY
	}
}

/**
 * Generates boxes for a given route and distance
 *
 * @param {LatLng[]} vertices The vertices of the path over which to lay the grid
 * @param {Number} range The spacing of the grid cells.
 */
func (r *RouteBoxer) buildGrid() {
	// Create a Bound object that contains the whole path
	routeBounds := r.vertices.Bound()

	// Finds the center of the bounding box
	boundCenter := routeBounds.Center()

	// Starting from the center define grid lines outwards vertically until they
	//  extend beyond the edge of the bounding box by more than one cell
	r.latGrid = append(r.latGrid, boundCenter.Lat())

	// Add lines from the center out to the north
	r.latGrid = append(r.latGrid, RhumbDestinationPoint(*boundCenter, 0, r.distanceRange).Lat())

	for i := 2; r.latGrid[i-2] < routeBounds.NorthEast().Lat(); i++ {
		r.latGrid = append(r.latGrid, RhumbDestinationPoint(*boundCenter, 0, r.distanceRange*float64(i)).Lat())
	}

	// Add lines from the center out to the south
	for i1 := 1; r.latGrid[1] > routeBounds.SouthWest().Lat(); i1++ {
		r.latGrid = append([]float64{RhumbDestinationPoint(*boundCenter, 180, r.distanceRange*float64(i1)).Lat()}, r.latGrid...)
	}

	r.lngGrid = append(r.lngGrid, boundCenter.Lng())

	// Add lines from the center out to the east
	r.lngGrid = append(r.lngGrid, RhumbDestinationPoint(*boundCenter, 90, r.distanceRange).Lng())
	for i2 := 2; r.lngGrid[i2-2] < routeBounds.NorthEast().Lng(); i2++ {
		r.lngGrid = append(r.lngGrid, RhumbDestinationPoint(*boundCenter, 90, r.distanceRange*float64(i2)).Lng())
	}

	// Add lines from the center out to the west
	for i3 := 1; r.lngGrid[1] > routeBounds.SouthWest().Lng(); i3++ {
		//	this._lngGrid.Insert(0, routeBoundsCenter.RhumbDestinationPoint(270, range * i3).Lng);
		r.lngGrid = append([]float64{RhumbDestinationPoint(*boundCenter, 270, r.distanceRange*float64(i3)).Lng()}, r.lngGrid...)
	}

	r.grid = make([][]int, len(r.lngGrid))
	for i := 0; i < len(r.lngGrid); i++ {
		r.grid[i] = make([]int, len(r.latGrid))
	}
}

/**
 * Find all of the cells in the overlaid grid that the path intersects
 *
 * @param {LatLng[]} vertices The vertices of the path
 */
func (r *RouteBoxer) FindIntersectingCells() {
	// Find the cell where the path begins
	hintXY := r.getCellCoords(r.vertices[0])

	// Mark that cell and it's neighbours for inclusion in the boxes
	r.markCell(hintXY)

	// Work through each vertex on the path identifying which grid cell it is in
	for i := 1; i < len(r.vertices); i++ {
		// Use the known cell of the previous vertex to help find the cell of this vertex
		gridXY := r.getGridCoordsFromHint(r.vertices[i], r.vertices[i-1], hintXY)

		if gridXY[0] == hintXY[0] && gridXY[1] == hintXY[1] {
			// This vertex is in the same cell as the previous vertex
			// The cell will already have been marked for inclusion in the boxes
			continue
		} else if (hintXY[0]-gridXY[0] == 1 && hintXY[1] == gridXY[1]) ||
			(hintXY[0] == gridXY[0] && hintXY[1]-gridXY[1] == 1) {
			// This vertex is in a cell that shares an edge with the previous cell
			// Mark this cell and it's neighbours for inclusion in the boxes
			r.markCell(gridXY)
		} else {
			// This vertex is in a cell that does not share an edge with the previous
			//  cell. This means that the path passes through other cells between
			//  this vertex and the previous vertex, and we must determine which cells
			//  it passes through
			r.getGridIntersects(r.vertices[i-1], r.vertices[i], hintXY, gridXY)
		}

		// Use this cell to find and compare with the next one
		hintXY = gridXY
	}
}

/**
 * Find the cell a path vertex is in based on the known location of a nearby
 *  vertex. This saves searching the whole grid when working through vertices
 *  on the polyline that are likely to be in close proximity to each other.
 *
 * @param {LatLng[]} latlng The latlng of the vertex to locate in the grid
 * @param {LatLng[]} hintlatlng The latlng of the vertex with a known location
 * @param {Number[]} hint The cell containing the vertex with a known location
 * @return {Number[]} The cell coordinates of the vertex to locate in the grid
 */
func (r *RouteBoxer) getGridCoordsFromHint(latlng geo.Point, hintlatlng geo.Point, hint []int) []int {
	var x, y int
	if latlng.Lng() > hintlatlng.Lng() {
		for x = hint[0]; r.lngGrid[x+1] < latlng.Lng(); x++ {
		}
	} else {
		for x = hint[0]; r.lngGrid[x] > latlng.Lng(); x-- {
		}
	}

	if latlng.Lat() > hintlatlng.Lat() {
		for y = hint[1]; r.latGrid[y+1] < latlng.Lat(); y++ {
		}
	} else {
		for y = hint[1]; r.latGrid[y] > latlng.Lat(); y-- {
		}
	}

	return ([]int{x, y})
}

/**
 * Find the cell a path vertex is in by brute force iteration over the grid
 *
 * @param {LatLng[]} latlng The latlng of the vertex
 * @return {Number[][]} The cell coordinates of this vertex in the grid
 */
func (r *RouteBoxer) getCellCoords(latlng geo.Point) []int {
	var x, y int
	for x = 0; r.lngGrid[x] < latlng.Lng(); x++ {
	}
	for y = 0; r.latGrid[y] < latlng.Lat(); y++ {
	}
	result := []int{x - 1, y - 1}
	return result
}

/**
 * Mark a cell and the 8 immediate neighbours for inclusion in the boxes
 *
 * @param {Number[]} square The cell to mark
 */
func (r *RouteBoxer) markCell(cell []int) {
	var x = cell[0]
	var y = cell[1]
	r.grid[x-1][y-1] = 1
	r.grid[x][y-1] = 1
	r.grid[x+1][y-1] = 1
	r.grid[x-1][y] = 1
	r.grid[x][y] = 1
	r.grid[x+1][y] = 1
	r.grid[x-1][y+1] = 1
	r.grid[x][y+1] = 1
	r.grid[x+1][y+1] = 1
}

/**
 * Identify the grid squares that a path segment between two vertices
 * intersects with by:
 * 1. Finding the bearing between the start and end of the segment
 * 2. Using the delta between the lat of the start and the lat of each
 *    latGrid boundary to find the distance to each latGrid boundary
 * 3. Finding the lng of the intersection of the line with each latGrid
 *     boundary using the distance to the intersection and bearing of the line
 * 4. Determining the x-coord on the grid of the point of intersection
 * 5. Filling in all squares between the x-coord of the previous intersection
 *     (or start) and the current one (or end) at the current y coordinate,
 *     which is known for the grid line being intersected
 *
 * @param {LatLng} start The latlng of the vertex at the start of the segment
 * @param {LatLng} end The latlng of the vertex at the end of the segment
 * @param {Number[]} startXY The cell containing the start vertex
 * @param {Number[]} endXY The cell containing the vend vertex
 */
func (r *RouteBoxer) getGridIntersects(start geo.Point, end geo.Point, startXY []int, endXY []int) {
	var edgePoint geo.Point
	var edgeXY []int
	var i int

	brng := RhumBearingTo(start, end)

	hint := start
	hintXY := startXY

	if end.Lat() > start.Lat() {
		// Iterate over the east to west grid lines between the start and end cells
		for i = startXY[1] + 1; i <= endXY[1]; i++ {
			// Find the latlng of the point where the path segment intersects with
			//  this grid line (Step 2 & 3)
			edgePoint = r.getGridIntersect(start, brng, r.latGrid[i])

			// Find the cell containing this intersect point (Step 4)
			edgeXY = r.getGridCoordsFromHint(edgePoint, hint, hintXY)

			// Mark every cell the path has crossed between this grid and the start,
			//   or the previous east to west grid line it crossed (Step 5)
			r.fillInGridSquares(hintXY[0], edgeXY[0], i-1)

			// Use the point where it crossed this grid line as the reference for the
			//  next iteration
			hint = edgePoint
			hintXY = edgeXY
		}

		// Mark every cell the path has crossed between the last east to west grid
		//  line it crossed and the end (Step 5)
		r.fillInGridSquares(hintXY[0], endXY[0], i-1)
	} else {
		// Iterate over the east to west grid lines between the start and end cells
		for i = startXY[1]; i > endXY[1]; i-- {
			// Find the latlng of the point where the path segment intersects with
			//  this grid line (Step 2 & 3)
			edgePoint = r.getGridIntersect(start, brng, r.latGrid[i])

			// Find the cell containing this intersect point (Step 4)
			edgeXY = r.getGridCoordsFromHint(edgePoint, hint, hintXY)

			// Mark every cell the path has crossed between this grid and the start,
			//   or the previous east to west grid line it crossed (Step 5)
			r.fillInGridSquares(hintXY[0], edgeXY[0], i)

			// Use the point where it crossed this grid line as the reference for the
			//  next iteration
			hint = edgePoint
			hintXY = edgeXY
		}

		// Mark every cell the path has crossed between the last east to west grid
		//  line it crossed and the end (Step 5)
		r.fillInGridSquares(hintXY[0], endXY[0], i)
	}
}

/**
 * Mark all cells in a given row of the grid that lie between two columns
 *   for inclusion in the boxes
 *
 * @param {Number} startx The first column to include
 * @param {Number} endx The last column to include
 * @param {Number} y The row of the cells to include
 */
func (r *RouteBoxer) fillInGridSquares(startx int, endx int, y int) {
	var x int
	if startx < endx {
		for x = startx; x <= endx; x++ {
			cell := []int{x, y}
			r.markCell(cell)
		}
	} else {
		for x = startx; x >= endx; x-- {
			cell := []int{x, y}
			r.markCell(cell)
		}
	}
}

/**
 * Create two sets of bounding boxes, both of which cover all of the cells that
 *   have been marked for inclusion.
 *
 * The first set is created by combining adjacent cells in the same column into
 *   a set of vertical rectangular boxes, and then combining boxes of the same
 *   height that are adjacent horizontally.
 *
 * The second set is created by combining adjacent cells in the same row into
 *   a set of horizontal rectangular boxes, and then combining boxes of the same
 *   width that are adjacent vertically.
 *
 */
func (r *RouteBoxer) mergeIntersectingCells() {
	x, y := 0, 0

	var box, currentBox *geo.Bound

	for y = 0; y < len(r.grid[0]); y++ {
		for x = 0; x < len(r.grid); x++ {
			if r.grid[x][y] == 1 {
				// This cell is marked for inclusion. If the previous cell in this
				//   row was also marked for inclusion, merge this cell into it's box.
				// Otherwise start a new box.
				box = r.getCellBounds([]int{x, y})
				if currentBox != nil {
					currentBox.Extend(box.NorthEast())
				} else {
					currentBox = box
				}
			} else {
				// This cell is not marked for inclusion. If the previous cell was
				//  marked for inclusion, merge it's box with a box that spans the same
				//  columns from the row below if possible.
				r.mergeBoxesY(currentBox)
				currentBox = nil
			}
		}
		r.mergeBoxesY(currentBox)
		currentBox = nil
	}

	// Traverse the grid a column at a time
	for x = 0; x < len(r.grid); x++ {
		for y = 0; y < len(r.grid[0]); y++ {
			if r.grid[x][y] == 1 {
				// This cell is marked for inclusion. If the previous cell in this
				//   column was also marked for inclusion, merge this cell into it's box.
				// Otherwise start a new box.
				cell := []int{x, y}
				if currentBox != nil {
					box = r.getCellBounds(cell)
					currentBox.Extend(box.NorthEast())
				} else {
					currentBox = r.getCellBounds(cell)
				}
			} else {
				// This cell is not marked for inclusion. If the previous cell was
				//  marked for inclusion, merge it's box with a box that spans the same
				//  rows from the column to the left if possible.
				r.mergeBoxesX(currentBox)
				currentBox = nil
			}
		}
		// If the last cell was marked for inclusion, merge it's box with a matching
		//  box from the column to the left if possible.
		r.mergeBoxesX(currentBox)
		currentBox = nil
	}
}

/**
 * Search for an existing box in an adjacent row to the given box that spans the
 * same set of columns and if one is found merge the given box into it. If one
 * is not found, append this box to the list of existing boxes.
 *
 * @param {LatLngBounds}  The box to merge
 */
func (r *RouteBoxer) mergeBoxesX(box *geo.Bound) {
	if box != nil {
		for i := 0; i < len(r.boxesX); i++ {
			if math.Abs(r.boxesX[i].NorthEast().Lng()-box.SouthWest().Lng()) < 0.001 &&
				math.Abs(r.boxesX[i].SouthWest().Lat()-box.SouthWest().Lat()) < 0.001 &&
				math.Abs(r.boxesX[i].NorthEast().Lat()-box.NorthEast().Lat()) < 0.001 {
				r.boxesX[i].Extend(box.NorthEast())
				return
			}
		}
		r.boxesX = append(r.boxesX, *box)
	}
}

/**
 * Search for an existing box in an adjacent column to the given box that spans
 * the same set of rows and if one is found merge the given box into it. If one
 * is not found, append this box to the list of existing boxes.
 *
 * @param {LatLngBounds}  The box to merge
 */
func (r *RouteBoxer) mergeBoxesY(box *geo.Bound) {
	if box != nil {
		for i := 0; i < len(r.boxesY); i++ {
			if math.Abs(r.boxesY[i].NorthEast().Lat()-box.SouthWest().Lat()) < 0.001 &&
				math.Abs(r.boxesY[i].SouthWest().Lng()-box.SouthWest().Lng()) < 0.001 &&
				math.Abs(r.boxesY[i].NorthEast().Lng()-box.NorthEast().Lng()) < 0.001 {
				r.boxesY[i].Extend(box.NorthEast())
				return
			}
		}
		r.boxesY = append(r.boxesY, *box)
	}
}

func (r *RouteBoxer) getCellBounds(cell []int) *geo.Bound {
	return geo.NewBoundFromPoints(
		geo.NewPointFromLatLng(r.latGrid[cell[1]], r.lngGrid[cell[0]]),
		geo.NewPointFromLatLng(r.latGrid[cell[1]+1], r.lngGrid[cell[0]+1]),
	)
}

func (r *RouteBoxer) getGridIntersect(start geo.Point, brng float64, gridLineLat float64) geo.Point {
	d := (geo.EarthRadius / 1000) * ((deg2rad(gridLineLat) - deg2rad(start.Lat())) / math.Cos(deg2rad(brng)))
	return *RhumbDestinationPoint(start, brng, d)
}

func (r *RouteBoxerResult) ToGeoJson() GeoJsonResult {
	polygons := [][][]geo.Point{}
	for _, box := range *r {
		polygon := [][]geo.Point{
			{
				{box.NorthWest().Lng(), box.NorthWest().Lat()},
				{box.NorthEast().Lng(), box.NorthEast().Lat()},
				{box.SouthEast().Lng(), box.SouthEast().Lat()},
				{box.SouthWest().Lng(), box.SouthWest().Lat()},
				{box.NorthWest().Lng(), box.NorthWest().Lat()},
			},
		}
		polygons = append(polygons, polygon)
	}

	result := GeoJsonResult{"MultiPolygon", polygons}
	return result
}
