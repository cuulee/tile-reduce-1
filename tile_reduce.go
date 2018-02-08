package tile_reduce

import (
	"github.com/murphy214/mbtiles-util"
	m "github.com/murphy214/mercantile"
	"fmt"
	"time"
	"github.com/paulmach/go.geojson"
	"math/rand"

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


// A structure to hold all the channels
type Channels struct {
	Guard chan struct{}	
	TileID_Chan chan m.TileID
	Raw_Chan chan Raw_Struct 
	VT_Chan chan VT_Struct
	Return_Chan chan Return_Struct
}


func New_Channels() Channels {
	buffer_size := 10000
	if buffer_size != 0 {
		guard := make(chan struct{},1000)
		tilechan := make(chan m.TileID,buffer_size)
		rawchan := make(chan Raw_Struct,buffer_size)
		parsechan := make(chan VT_Struct,buffer_size)
		returnchan := make(chan Return_Struct,buffer_size)
		return Channels{
			Guard:guard,
			TileID_Chan:tilechan,
			Raw_Chan:rawchan,
			VT_Chan:parsechan,
			Return_Chan:returnchan,
		}

	} else {
		guard := make(chan struct{},1000)
		tilechan := make(chan m.TileID)
		rawchan := make(chan Raw_Struct)
		parsechan := make(chan VT_Struct)
		returnchan := make(chan Return_Struct)		

		return Channels{
			Guard:guard,
			TileID_Chan:tilechan,
			Raw_Chan:rawchan,
			VT_Chan:parsechan,
			Return_Chan:returnchan,
		}
	}

	return Channels{}
}


// the raw byte away seqence returned
type Raw_Struct struct {
	TileID m.TileID
	Bytes []byte
}

// the layermap geojson representation returned
type VT_Struct struct {
	TileID m.TileID
	LayerMap map[string][]*geojson.Feature
}

// the value following the mapped function
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
	Channels Channels
	Processes int // number of processes
	TotalCount int // total count a value used internally
	CurrentPos int // the current pos in tiles
	CollectPos int // collect position
	StartBool bool // the starting bool to ensure blocking
}

type Reduce_Func func(k m.TileID,v map[string][]*geojson.Feature) interface{}

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
		fmt.Println(maxz)
		tile_reduce.Tiles = BoundingBox_Tiles(tile_reduce.BoundingBox,minz,maxz)		
		tile_reduce.TotalCount = len(tile_reduce.Tiles)		
	} else if tile_reduce.Type == Tiles {
		tile_reduce.TotalCount = len(tile_reduce.Tiles)				
	}
}


func (tile_reduce *Tile_Reduce_Config) Shuffle_Tiles() {
	dest := make([]m.TileID, len(tile_reduce.Tiles))
	perm := rand.Perm(len(tile_reduce.Tiles))
	for i, v := range perm {
	    dest[v] = tile_reduce.Tiles[i]
	}
	tile_reduce.Tiles = dest
}
// Masks function
func Mask(data interface{}) interface{} {
	return data
}


func Get_Tile_Splits(num_tiles int,num int) [][2]int {
	delta := num_tiles / num 
	current := 0
	newlist := []int{current}
	for current < num_tiles {
		current += delta
		newlist = append(newlist,current)
	}

	if newlist[len(newlist)-1] > num_tiles {
		newlist[len(newlist)-1] = num_tiles
	}


	//
	newlist2 := []int{}
	for pos,i := range newlist {
		if pos == 0 || pos == len(newlist) - 1 {
			newlist2 = append(newlist2,i)
		} else {
			newlist2 = append(newlist2,[]int{i,i}...)
		}
	}
	newlist = newlist2
	ind_list := make([][2]int,len(newlist)/2)

	for i := 0; i < len(newlist); i+= 2 {
		ind_list[i/2] = [2]int{newlist[i],newlist[i+1]}
	}
	return ind_list
}


func (tile_reduce *Tile_Reduce_Config) Tile_Reduce(reduce_func Reduce_Func) {
	// getting number of channels
	tile_reduce.Channels = New_Channels()

	// getting the list of tiles
	tile_reduce.Get_Tiles()
	tile_reduce.Shuffle_Tiles()
	//tile_reduce.Tiles = tile_reduce.Tiles[:10000]
	//tile_reduce.TotalCount = 10000

	// getting splits 
	//ind_pos := Get_Tile_Splits(len(tile_reduce.Tiles),tile_reduce.Processes)
	go tile_reduce.Start_Streams()
	// sending into go functions
	for i := 0; i < tile_reduce.Processes; i++  {
		//ind1,ind2 := ind[0],ind[1]
		fmt.Printf("Created %d worker.\n",i)
		go tile_reduce.Worker(reduce_func)
	}
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
 

func (tile_reduce *Tile_Reduce_Config) Worker(reduce_func Reduce_Func) {
	for tile_reduce.CollectPos <= tile_reduce.TotalCount {
		vtstruct := <-tile_reduce.Channels.VT_Chan
		//fmt.Println(vtstruct)
		tile_reduce.Channels.Return_Chan <- Return_Struct{
			TileID:vtstruct.TileID,
			Interface:
				reduce_func(vtstruct.TileID, 
					vtstruct.LayerMap, // the raw byte array
					),
			}
	}	
}