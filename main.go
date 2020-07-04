package main

import (
	"fmt"

	"github.com/perlin-network/noise"
)

func main() {
	me, err := noise.NewNode(nodeOpts...)
	if err != nil {
		panic(err)
	}
	defer me.Close()
	if err := me.Listen(); err != nil {
		panic(err)
	}
	fmt.Println(me.Addr())
}
