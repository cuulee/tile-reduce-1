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
	"github.com/murphy214/tile-reduce"
	mbutil "github.com/murphy214/mbtiles-util"
	m "github.com/murphy214/mercantile"
	"fmt"
)


func main() {
	// getting mbtile object (our source)
	mbtile := mbutil.Read_Mbtiles("america.mbtiles")

	// creating the tile reduce structure that will map a single function
	tr := tile_reduce.Tile_Reduce_Config{
		Source:&mbtile,
		Processes:1000, // 1000 concurrent processes at time 
		Type:tile_reduce.All,
	}

	// defining funciton to map
	mapfunc := func(k m.TileID,bytes []byte) interface{} {
		// convering into featuers
		// note this function is specifically used for
		// mapbox qa tiles because of their geom encoding
		layermap := mbutil.Convert_Vt_Bytes_QA(bytes,k) 

		// iterating through each layer getting the total features
		num_feats := 0
		for _,features := range layermap {
			num_feats += len(features)
		}
		
		return tile_reduce.Mask(num_feats)
	}
	
	// sending go function into concurrentcy land

	go tr.Tile_Reduce(mapfunc)
	
	// code blocks here
	total_number_features := 0
	count := 0
	for tr.Next() {
		val := <-tr.Channel
		total_number_features += val.Interface.(int)
		count += 1
		fmt.Printf("\r[%d/%d] Tiles Completed",count,tr.TotalCount)
	}

	fmt.Printf("\n\nTotal Number of Features: %d",total_number_features)

}
```

# Output 
![](https://user-images.githubusercontent.com/10904982/35489718-693345c8-0467-11e8-893f-cff74a4090c4.png)
