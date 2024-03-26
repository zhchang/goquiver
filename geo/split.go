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

func splitPolygon(p *geom.Polygon, ps *[]geom.Polygon, lc loopContext) {
	if len(p.Coordinates()) < 200 || lc.curDepth >= lc.maxDepth {
		*ps = append(*ps, *p)
		return
	}
	bb := p.Envelope()
	_min, _ := bb.Min().XY()
	_max, _ := bb.Max().XY()
	longerEdge := max(_max.X-_min.X, _max.Y-_min.Y)
	unit := longerEdge / float64(lc.clusterSize)
	rects := []geom.Polygon{}
	for x := _min.X; x < _max.X; x += unit {
		for y := _min.Y; y < _max.Y; y += unit {
			x0 := round(x)
			y0 := round(y)
			x1 := round(min(_max.X + unit))
			y1 := round(min(_max.Y, y+unit))
			coords := []float64{x0, y0, x0, y1, x1, y1, x1, y0, x0, y0}
			rects = append(rects, geom.NewPolygon([]geom.LineString{geom.NewLineString(geom.NewSequence(coords, geom.DimXY))}))
		}
	}
	var err error
	var g geom.Geometry
	for _, rect := range rects {
		if g, err = geom.Intersection(p.AsGeometry(), rect.AsGeometry()); err != nil {
			continue
		}
		var pi geom.Polygon
		var ok bool
		if pi, ok = g.AsPolygon(); !ok {
			continue
		}
		splitPolygon(&pi, ps, loopContext{lc.maxDepth, lc.curDepth + 1, lc.clusterSize})
	}
	//return
}
