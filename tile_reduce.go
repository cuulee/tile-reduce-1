package tile_reduce

import (
	"github.com/murphy214/mbtiles-util"
	m "github.com/murphy214/mercantile"
	"fmt"
	"time"
	"github.com/paulmach/go.geojson"
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
	Feature *geojson.Feature // a geojson feature (line or polygon) to use as an area to analyze
	BoundingBox m.Extrema // bounding box given for mapping about a boudning box
	Source *mbutil.Mbtiles // source given this should become sources
	Channel chan Return_Struct // channel for the return structure
	Processes int // number of processes
	TotalCount int // total count a value used internally
	Guard chan struct{}
	CurrentPos int // the current pos in tiles
	CollectPos int // collect position
	StartBool bool // the starting bool to ensure blocking
}

type Reduce_Func func(k m.TileID,v []byte) interface{}

// gets the tiles at a given zoom level 
func (tile_reduce *Tile_Reduce_Config) Get_Tiles() {
	if tile_reduce.Type == Zoom {
		rows,err := tile_reduce.Source.Tx.Query("SELECT tile_column,tile_row,zoom_level FROM tiles where zoom_level = ?",tile_reduce.Zoom)
		if err != nil {
			fmt.Println(err)
		}

		// appending all tiles
		for rows.Next() {
			var x,y,z int

			rows.Scan(&x,&y,&z)
			y = (1 << uint64(z)) - y - 1
			tileid := m.TileID{int64(x),int64(y),uint64(z)}
			tile_reduce.Tiles = append(tile_reduce.Tiles,tileid)
		}
		tile_reduce.TotalCount = len(tile_reduce.Tiles)

	} else if tile_reduce.Type == All {
		rows,err := tile_reduce.Source.Tx.Query("SELECT tile_column,tile_row,zoom_level FROM tiles")
		if err != nil {
			fmt.Println(err)
		}

		// appending all tiles
		for rows.Next() {
			var x,y,z int

			rows.Scan(&x,&y,&z)
			y = (1 << uint64(z)) - y - 1
			tileid := m.TileID{int64(x),int64(y),uint64(z)}
			tile_reduce.Tiles = append(tile_reduce.Tiles,tileid)
		}
		tile_reduce.TotalCount = len(tile_reduce.Tiles)

	} else if tile_reduce.Type == Feature {
		minz := tile_reduce.Source.MinZoom
		maxz := tile_reduce.Source.MaxZoom
		tile_reduce.Tiles = Feature_Tiles(tile_reduce.Feature,minz,maxz)
		tile_reduce.TotalCount = len(tile_reduce.Tiles)
	} else if tile_reduce.Type == BoundingBox {
		// boundingbox code here
		minz := tile_reduce.Source.MinZoom
		maxz := tile_reduce.Source.MaxZoom
		tile_reduce.Tiles = BoundingBox_Tiles(tile_reduce.BoundingBox,minz,maxz)		
		tile_reduce.TotalCount = len(tile_reduce.Tiles)		
	} else if tile_reduce.Type == Tiles {
		tile_reduce.TotalCount = len(tile_reduce.Tiles)				
	}
}

type Temp_Struct struct {
	TileID m.TileID
	Bytes []byte
} 

// gets the next map for a chunk of tiles
func (tile_reduce *Tile_Reduce_Config) Next_Map() map[m.TileID][]byte {
	var delta int
	if tile_reduce.TotalCount <= tile_reduce.CurrentPos + tile_reduce.Processes {
		delta = tile_reduce.TotalCount - tile_reduce.CurrentPos
	} else {
		delta = tile_reduce.Processes
	}
	// getting new current
	newcurrent := tile_reduce.CurrentPos + delta

	// getting temporary tiles
	temptiles := tile_reduce.Tiles[tile_reduce.CurrentPos:newcurrent]

	// setting the current pos to the new current
	tile_reduce.CurrentPos = newcurrent

	// creating channel and sending into go function
	c := make(chan Temp_Struct)
	for _,i := range temptiles {
		go func(i m.TileID,c chan Temp_Struct) {
			c <- Temp_Struct{TileID:i,Bytes:tile_reduce.Source.Query(i)}
		}(i,c)
	}

	tempmap := map[m.TileID][]byte{}
	// collecting the channel and adding to map
	for range temptiles {
		val := <-c
		tempmap[val.TileID] = val.Bytes
	}
	return tempmap

}
	
// mainline tile reduce function
func (tile_reduce *Tile_Reduce_Config) Tile_Reduce(reduce_func Reduce_Func) {
	// setting total count to trigger code blocking
	// some race conditions could result if we don't set this 
	// value immediately
	// setting up blocking channels
	tile_reduce.Channel = make(chan Return_Struct)
	tile_reduce.Guard = make(chan struct{}, tile_reduce.Processes)

	// getting the list of tiles
	tile_reduce.Get_Tiles()

	// a next function for when weve hit the end of our source
	for tile_reduce.NextMap() {
		// getting tilemap
		mymap := tile_reduce.Next_Map()
		// iterating through block of tiles
		// this will get more complex
		for k,v := range mymap {
			// adding to total count and sending blocking guard
			// this is a hacky way to ensure the blocking collector is instanted.
			tile_reduce.Guard <- struct{}{}
			
			// doing go func
			go func(k m.TileID,v []byte) {

				<-tile_reduce.Guard

				data := reduce_func(k,v)

				tile_reduce.Channel <- Return_Struct{TileID:k,Interface:data}


			}(k,v)
		}
	}

}

// boilerplate next function
func (tile_reduce *Tile_Reduce_Config) NextMap() bool {
	return (tile_reduce.CurrentPos == tile_reduce.TotalCount) == false
}

// next function for ensuring code blocks correctly
func (tile_reduce *Tile_Reduce_Config) Next() bool {
	var boolval bool
	tile_reduce.CollectPos += 1
	if tile_reduce.StartBool == false {
		tile_reduce.StartBool = true
		boolval = true
		time.Sleep(time.Millisecond*1)
	} else {
		boolval = tile_reduce.CollectPos <= tile_reduce.TotalCount

	}
	//fmt.Println(boolval,tile_reduce.CollectPos)
	return boolval
}


// Masks function
func Mask(data interface{}) interface{} {
	return data
}