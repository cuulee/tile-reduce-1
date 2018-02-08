package tile_reduce

import (
	m "github.com/murphy214/mercantile"
	vt "github.com/murphy214/vector-tile-go"

)


// this functions will start the stream of channeels that 
// will hopefully result in VT_Chans blocking at the bottum
// tileidchan <- tileid 
// 
func (tile_reduce *Tile_Reduce_Config) Start_Streams() {
	// starting both the consumers that collect down stream
	go tile_reduce.Stream_TileID()
	go tile_reduce.Stream_Raw()

	for _,tile := range tile_reduce.Tiles {
		tile_reduce.Channels.Guard <- struct{}{}
		go tile_reduce.Pass_TileID(tile)
	}
}

// pases the tileid to a channel
func (tile_reduce *Tile_Reduce_Config) Pass_TileID(tile m.TileID) {
	tile_reduce.Channels.TileID_Chan <- tile
}

// this function simply collects tileids and sends on to next process
// basically it continually sends tileids streaming in downstream to other processes
func (tile_reduce *Tile_Reduce_Config) Stream_TileID() {
	for i := 0; i < tile_reduce.TotalCount; i++ {
		tileid := <- tile_reduce.Channels.TileID_Chan
		go tile_reduce.Process_TileID(tileid)
	}
}

// pases the tileid to a channel
func (tile_reduce *Tile_Reduce_Config) Process_TileID(tile m.TileID) {
	rawstruct := Raw_Struct{ 
		TileID:tile,
		Bytes:tile_reduce.Source.Query(tile),
	}
	tile_reduce.Channels.Raw_Chan <- rawstruct
}

// this function simply collects tileids and sends on to next process
// basically it continually sends raw_structs streaming in downstream to other processes
func (tile_reduce *Tile_Reduce_Config) Stream_Raw() {
	for i := 0; i < tile_reduce.TotalCount; i++ {
		rawstruct := <- tile_reduce.Channels.Raw_Chan
		go tile_reduce.Process_Raw(rawstruct)
	}
}

// pases the tileid to a channel
func (tile_reduce *Tile_Reduce_Config) Process_Raw(raw Raw_Struct) {
	val := VT_Struct{
		LayerMap:vt.ToGeoJSON(raw.Bytes,raw.TileID),
		TileID:raw.TileID,
	}
	tile_reduce.Channels.VT_Chan <- val
}

