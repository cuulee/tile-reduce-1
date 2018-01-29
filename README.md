# tile-reduce

# What is it?

Often times in geospatial processing we are doing things about a specific tile, and have implemented 100s of go routines methods or bits of codes doing this across many modules, because I thought go type system wouldn't allow for the mapping of functions to each tile gracefully. With a few quirks, this package does this saving hours of work and tons of code in the process. (hopefully)

So you can sort of think of this as a job scheduler of some arbitary function return an interface (which can then be cast to its original type down stream in collection or just returned as an empty interface and all i/o is done within the function to another file or something. This allows you to sort of chain these processes together in a beneficial manner as well. 

# Structure 

Currently tile-reduce supports 4 (well 5) mapping types being what tiles we actually operate on in our datasets. The mapping types being: 

1. **All** - all tiles in source
2. **Zoom** - all tiles in a source at a specific zoom
3. **Tiles** - given a set of tiles operate on all in source
4. **BoundingBox** - given a bounding box, find the smallest zoom that fits an entire tile within it, then envelope that bounding box with said zoom_level tiles finally drilling down to the max zoom of the data set 
5. **Feature** - given a feature cover the feature with the smallest zoom that fits an entire tile within it then do as the BoundingBox above

These mappings are fairly similiar to mapbox's node tile reduce. Examples of implementing each of these will be shown in examples later. 

# Concerns 

I'm not fully aware of variable scopes in go it would take me a second to figure it out but I do its possible to float globals into functions depending on either how you define the variable or how you define the function. In other words code may have to be written a certain way to ensure that variables that are needed during processing say another dataset or anything structure can be used without being explicity defined in the function signature. 

Currently this implementation supports one source file however, I have plans for multiple sources the issue is do I try to melt one struct and all these functions to take both one source and multiple or duplicate a lot of code and an entirely new struct for the sources even though much is it entirely the same. 

Currently ALL and ZOOM mapping types ingest every tileid into memory as to account for when we have multiple sources and free us from the task of maintaining the different tiles between sources instead one big tile list. However at size 20 zoom their are billions of tiles, and at some point we'll need to push the tiles out of memory. I was thinking a memory mapped 4,4,1 byte (int32,int32,byte representation of zoom i.e. int8) implementation of encoding the ids to a file and reading each tile along the way or blocks of tiles. It would be just as easy as csv encoding, in fact  a lot more structure. 

For example say were doing 500 processes at once 500 * 9 bytes 4500 bytes read 4500 at starting pos x increment starting pos, jump 9 bytes at a type mapping a simple byte encoding to integers. This will need to be done for sure, I'm just not sure if int32 is large enough. 

```golang 
package main

import (
	"github.com/paulmach/go.geojson"
	"io/ioutil"
	"github.com/murphy214/mbtiles-util"
	m "github.com/murphy214/mercantile"
	"github.com/murphy214/tile-reduce"

)

// this function returns all the features within layers in a given vector tile  
// a geojson feature array 
func Mapped_Func(k m.TileID,v []byte) interface{} {
	// gettiing the map of each layer
	a := mbutil.Convert_Vt_Bytes(v,k)
	
	newfeats := []*geojson.Feature{}
	// iterating through each layer
	for _,vv := range a {	
		newfeats = append(newfeats,vv...) // appending the set of features to total list
	}
	return tile_reduce.Mask(newfeats) // projecting the feature array to an interface 
}


func main() {
	// reading a mbtiles context
	a := mbutil.Read_Mbtiles("delaware.mbtiles")

	// creating the tile reduce structure that will map a single function
	tr := tile_reduce.Tile_Reduce_Config{Zoom:12,Source:&a,Processes:1000}
	
	// sending the tile_reduce function off to concurrency land
	go tr.Tile_Reduce(Mapped_Func)

	// code will block here as tiles output are collected as an interface
	count := 0
	newfeats := []*geojson.Feature{}
	for tr.Next(count) {
		val := <-tr.Channel // collecting value from channel
		feats := val.Interface // selects the interface as we don't need the tileid
		newfeats = append(newfeats,feats.([]*geojson.Feature)...) // appending the features after we project its typing
		count += 1
	}

	// writing to file
	fc := &geojson.FeatureCollection{Features:newfeats}
	stval,_ := fc.MarshalJSON()
	ioutil.WriteFile("a.geojson",[]byte(stval),0677)
}
```

# Output 
![](https://user-images.githubusercontent.com/10904982/35489718-693345c8-0467-11e8-893f-cff74a4090c4.png)
