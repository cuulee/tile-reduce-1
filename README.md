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
	vt "github.com/murphy214/vector-tile-go"
	m "github.com/murphy214/mercantile"
	"fmt"
	"time"
	"io/ioutil"
	"github.com/paulmach/go.geojson"
)

func M(k m.TileID,v []byte) map[string][]*geojson.Feature {
	return vt.New_Vector_Tile(v).ToGeoJSON(k)
}


func consume(k m.TileID) {

}

func main() {
	// getting mbtile object (our source)
	mbtile := mbutil.Read_Mbtiles("america.mbtiles")
	mbtile.MaxZoom = 12

	// creating the tile reduce structure that will map a single function
	tr := tile_reduce.Tile_Reduce_Config{
		Source:&mbtile,
		Processes:1000, // 1000 concurrent processes at time 
		Type:tile_reduce.All,
	}



	// defining funciton to map
	// this function uses the newly implemented vector-tile-go to 
	// read each feature property with out reading each feature entirely
	// so you have access to properties and then can then go feature.ToGeoJSON(tileid)
	mapfunc := func(k m.TileID,v []byte) interface{} {
		// convering into featuers
		// note this function is specifically used for
		// mapbox qa tiles because of their geom encoding
		num_feats := 0
		la := vt.New_Vector_Tile(v)
		for _,vv := range la {
			i := 0 
			for i < vv.Number_Features {
				vv.Feature(i)
				i++
			}
			num_feats += i
		}
		
		return tile_reduce.Mask(num_feats)
	}
	
	// sending go function into concurrentcy land
	s := time.Now()
	go tr.Tile_Reduce(mapfunc)
	
	// code blocks here
	total_number_features := 0
	count := 0
	feats := []*geojson.Feature{}

	for tr.Next() {
		val := <-tr.Channel
		total_number_features += val.Interface.(int)
		count += 1
		if count % 1000 == 1 { 
			fmt.Printf("\r[%d/%d] Tiles Completed, tiles / s: %f",count,tr.TotalCount,float64(total_number_features) / time.Now().Sub(s).Seconds())
		}
	}
	fc := &geojson.FeatureCollection{Features:feats}
	ss,_ := fc.MarshalJSON()
	ioutil.WriteFile("a.geojson",ss,0677)


	fmt.Printf("\n\nTotal Number of Features: %d in %s time.",total_number_features,time.Now().Sub(s))

}
```
