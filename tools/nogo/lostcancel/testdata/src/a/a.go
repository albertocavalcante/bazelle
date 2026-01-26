package a

import "context"

func bad() {
	ctx, _ := context.WithCancel(context.Background()) // want "the cancel function returned by context.WithCancel should be called, not discarded, to avoid a context leak"
	_ = ctx
}

func good() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = ctx
}
