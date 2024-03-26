package geo

import (
	"fmt"
	"math"

	"github.com/peterstace/simplefeatures/geom"
)

func Split(input []byte, depth, cluster int) ([]byte, error) {
	var err error
	var fc geom.GeoJSONFeatureCollection
	if err = fc.UnmarshalJSON(input); err != nil {
		return nil, err
	}
	var polygons = []geom.Polygon{}
	for _, f := range fc {
		g := f.Geometry
		switch g.Type() {
		case geom.TypePolygon:
			var p = g.MustAsPolygon()
			splitPolygon(&p, &polygons, loopContext{depth, 0, cluster})
		case geom.TypeMultiPolygon:
			var mp = g.MustAsMultiPolygon()
			for i := 0; i < mp.NumPolygons(); i++ {
				p := mp.PolygonN(i)
				splitPolygon(&p, &polygons, loopContext{3, 0, 3})
			}
		default:
			return nil, fmt.Errorf("invalid type to split: %s", g.Type())
		}
	}
	var gc geom.GeoJSONFeatureCollection
	for _, p := range polygons {
		var f geom.GeoJSONFeature
		f.Geometry = p.AsGeometry()
		gc = append(gc, f)
	}
	var output []byte
	if output, err = gc.MarshalJSON(); err != nil {
		return nil, err
	}
	return output, nil
}

type loopContext struct {
	maxDepth    int
	curDepth    int
	clusterSize int
}

func round(a float64) float64 {
	return math.Round(a*1e7) / 1e7
}

func getCoordCount(p *geom.Polygon) int64 {
	var r int64
	for _, c := range p.Coordinates() {
		r += int64(c.Length())
	}
	return r
}

/*
useful debug functions
func log(p *geom.Polygon, d int) {
	bb := p.Envelope()
	_min, _ := bb.Min().XY()
	_max, _ := bb.Max().XY()
	println("add poly", d, getCoordCount(p), fmt.Sprintf("%f,%f", _min.X, _min.Y), fmt.Sprintf("%f,%f", _max.X, _max.Y))
}

func testOverlap(ps *[]geom.Polygon, p *geom.Polygon) {
	for _, ep := range *ps {
		var ok bool
		if ok, _ = geom.Overlaps(ep.AsGeometry(), p.AsGeometry()); ok {
			//log(&ep)
			//log(p)
			panic("overlaps")
		}
		if ok, _ = geom.Contains(ep.AsGeometry(), p.AsGeometry()); ok {
			//log(&ep)
			//log(p)
			panic("contains")
		}
		if ok, _ = geom.Contains(p.AsGeometry(), ep.AsGeometry()); ok {
			//log(&ep)
			//log(p)
			panic("contains")
		}
	}
}
*/

func splitPolygon(p *geom.Polygon, ps *[]geom.Polygon, lc loopContext) {
	if getCoordCount(p) < 200 || lc.curDepth >= lc.maxDepth {
		//log(p, lc.curDepth)
		//testOverlap(ps, p)
		*ps = append(*ps, *p)
		return
	}
	bb := p.Envelope()
	if bb.IsEmpty() {
		return
	}
	var _min, _max geom.XY
	var full bool
	if _min, full = bb.Min().XY(); !full {
		return
	}
	if _max, full = bb.Max().XY(); !full {
		return
	}
	longerEdge := max(_max.X-_min.X, _max.Y-_min.Y)
	unit := longerEdge / float64(lc.clusterSize)
	rects := []geom.Polygon{}
	for x := _min.X; x < _max.X; x += unit {
		for y := _min.Y; y < _max.Y; y += unit {
			x0 := round(x)
			y0 := round(y)
			x1 := round(min(_max.X, x+unit))
			y1 := round(min(_max.Y, y+unit))
			if x0 == x1 || y0 == y1 {
				//skip bbox that has no area
				continue
			}
			coords := []float64{x0, y0, x0, y1, x1, y1, x1, y0, x0, y0}
			rect := geom.NewPolygon([]geom.LineString{geom.NewLineString(geom.NewSequence(coords, geom.DimXY))})
			//log(&rect, lc.curDepth)
			//testOverlap(&rects, &rect)
			rects = append(rects, rect)
		}
	}
	//*ps = append(*ps, rects...)
	//return
	var err error
	var g geom.Geometry
	var nlc = loopContext{lc.maxDepth, lc.curDepth + 1, lc.clusterSize}
	for _, rect := range rects {
		if g, err = geom.Intersection(p.AsGeometry(), rect.AsGeometry()); err != nil {
			continue
		}
		switch g.Type() {
		case geom.TypePolygon:
			pi := g.MustAsPolygon()
			splitPolygon(&pi, ps, nlc)
		case geom.TypeMultiPolygon:
			var mp = g.MustAsMultiPolygon()
			for i := 0; i < mp.NumPolygons(); i++ {
				pi := mp.PolygonN(i)
				splitPolygon(&pi, ps, nlc)
			}
		default:
			continue
		}
	}
	//return
}
