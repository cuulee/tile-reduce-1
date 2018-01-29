package tile_reduce 

import (
	m "github.com/murphy214/mercantile"
	tc "github.com/murphy214/tile-cover"
	"github.com/paulmach/go.geojson"
)

// drills all the additional tiles below the given tile
func Drill_Parent(tileid m.TileID,endzoom int) []m.TileID  {
	newlist := []m.TileID{}
	children := m.Children(tileid)
	c := make(chan []m.TileID)
	for _,child := range children {
		go func(child m.TileID,c chan []m.TileID) {
			if int(child.Z) < endzoom {
				c <- append(Drill_Parent(child,endzoom),child)
			} else if int(child.Z) == endzoom {
				c <- []m.TileID{child}
			} else {
				c <- []m.TileID{}
			}
		}(child,c)
	}

	for range children {
		newlist = append(newlist,<-c...)
	}

	return newlist
}

// expand tiles 
func Expand_Tiles(tiles []m.TileID,maxzoom int) []m.TileID {
	newtiles := []m.TileID{}
	for _,tile := range tiles {
		newtiles = append(newtiles,tile)
		newtiles = append(newtiles,Drill_Parent(tile,maxzoom)...)
	}	
	return newtiles
}

// gets the tiles related to a specific geojson feature
func Feature_Tiles(feature *geojson.Feature,minzoom int,maxzoom int) []m.TileID {
	tiles := tc.Tile_Cover(feature)
	tiles = Expand_Tiles(tiles,maxzoom)
	newtiles := []m.TileID{}
	for _,i := range tiles {
		if int(i.Z) >= minzoom {
			newtiles = append(newtiles,i)
		}
	}
	return newtiles		
}

// gets the tiles associated with a bounding box
func BoundingBox_Tiles(bds m.Extrema,minzoom int,maxzoom int) []m.TileID {
	// creating coords
	coords := [][][]float64{{{bds.E, bds.N},{bds.W, bds.N},{bds.W, bds.S},{bds.E, bds.S},{bds.E, bds.N}}}
	
	// creating feature
	feat := &geojson.Feature{Geometry:&geojson.Geometry{Type:"Polygon",Polygon:coords}}

	return Feature_Tiles(feat,minzoom,maxzoom)

}

