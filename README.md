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

# Example Usage

The example below reads from an americ (maybe north america) mapbox qa tilesets available for download as I figured it would be a good datum. The example not only serializes the vector tile but converts it into geojson features. (a step not really needed but worth showing) the qa tiles function assume gzip compression while the normal function just accepts the raw byte array in other words you would have to reflect the ungzip in your own mapfunc. 

The example was a ~5gb gzipped mbtiles file. On my machine (2012 MBP 15 in.) it completed 7m41.768725755s with 71,339,335 (71 million) thats > 100,000 features read / s from the worst state you can get the vector tiles in gzipped with QA encoding. 

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
