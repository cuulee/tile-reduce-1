# tile-reduce

# What is it?

This implementation is still pretty hacky as I haven't done much with the conifiguration YET and it doesn't support multiple sources or anything its suppose to. I've basically spent 90% of the time on this script tracking down race conditions for the collection part, but it works now! However, I'm sure I've done something not in an ideal manner. 

# How does it work?

Currently it has a pretty simple implementation and works a lot like mapbox's node tile-reduce the differences being the goes pretty strongly typed so we have to do a little bit of manipulation to within are mapped function to get it returned as a raw interface{} (which can be anything) from there I let the collection down stream be up to you. You can simply, have no collection at all and return a raw interface and modify files / sqlite stores within the function or do whatever you like this simply handles the mapping step. My functions instead of handling raw features handle raw byte arrays and you handle your sources as such, its a little more freedom but less structure as well. However, to get features at vector tiles you can simply use the mbutil.Convert_Vt_Bytes(bytes,tileid) to get out the features. 

# Caveats 

I've found a few bugs in my Convert_Vt_Bytes function that I need to track down related to features which shouldn't be to hard at all. However, a more pressing issue is supporting multi geometries in Convert_Vt_Bytes() which again shouldn't deviate to much from the normal implmentation but I've never implemented encoding of multi geometries and really I just want to be able to support other people or raw data sets downloaded. Although really it shouldn't be to hard. 

# A simple example 

The following shows an example of how to collect all the features of a vector tiles at a given zoom with the concurrent processes being set at 1000 and the zoom being set at 12. As you can see there isn't a terrible amount to the implementation however when more sources are added we may have to get a little more complex. 

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
