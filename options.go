package main

type Options struct {
	TargetFPS  int32
	DownScroll bool
}

var TheOptions Options = Options{
	TargetFPS:  60,
	DownScroll: false,
}