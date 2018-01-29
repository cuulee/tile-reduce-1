package tile_reduce

import (
	"github.com/murphy214/mbtiles-util"
	m "github.com/murphy214/mercantile"
	"fmt"
)

// map type
type Map_Type int

// denotes whether a specific tile_zoom will be used or a special one
const (
        All Map_Type = iota // all tiles will be mapped
        Zoom // all tiles will be mapped at a given zoom
       	Tiles // all tiles given via the Tiles config will be mapped
       	BoundingBox // all tiles given via a bounding box will be used
       	Feature // all tiles enveloped by a single geojson feature will be used 
)

type Return_Struct struct {
	TileID m.TileID
	Interface interface{}
}

// tile reduce configuration
type Tile_Reduce_Config struct {
	Zoom int // zoom given for mapping a given zoom 
	Type Map_Type // type given for a unique type
	Tiles []m.TileID // tiles given for mapping about tiles
	BoundingBox m.Extrema // bounding box given for mapping about a boudning box
	Source *mbutil.Mbtiles // source given this should become sources
	Channel chan Return_Struct // channel for the return structure
	Processes int // number of processes
	TotalCount int // total count a value used internally
	Guard chan struct{}
}

type Reduce_Func func(k m.TileID,v []byte) interface{}

// mainline tile reduce function
func (tile_reduce *Tile_Reduce_Config) Tile_Reduce(reduce_func Reduce_Func) {
	tile_reduce.Channel = make(chan Return_Struct)
	tile_reduce.Guard = make(chan struct{}, tile_reduce.Processes)

	// a next function for when weve hit the end of our source
	mymap := tile_reduce.Source.Chunk_Tiles_Zoom(tile_reduce.Processes,tile_reduce.Zoom)
	for tile_reduce.Source.Next() {
		// getting tilemap
		newtilemap := tile_reduce.Source.Chunk_Tiles_Zoom(tile_reduce.Processes,tile_reduce.Zoom)

		// iterating through block of tiles
		// this will get more complex
		for k,v := range mymap {
			// adding to total count and sending blocking guard
			// this is a hacky way to ensure the blocking collector is instanted.
			tile_reduce.TotalCount += 1
			tile_reduce.Guard <- struct{}{}
			
			// doing go func
			go func(k m.TileID,v []byte) {
				<-tile_reduce.Guard

				data := reduce_func(k,v)

				tile_reduce.Channel <- Return_Struct{TileID:k,Interface:data}


			}(k,v)
		}
		mymap = newtilemap

	}

}

// boilerplate next function
func (tile_reduce *Tile_Reduce_Config) Next(count int) bool {
	fmt.Printf("\r[%d/%d] Tiles collected!",count,tile_reduce.TotalCount)
	return tile_reduce.Source.Next() || (count < tile_reduce.TotalCount)
}

// Masks function
func Mask(data interface{}) interface{} {
	return data
}