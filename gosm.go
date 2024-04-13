package main

import (
	"golang.org/x/exp/constraints"
)

func f32[N constraints.Integer | constraints.Float](n N) float32{
	return float32(n)
}

func f64[N constraints.Integer | constraints.Float](n N) float64{
	return float64(n)
}

func i32[N constraints.Integer | constraints.Float](n N) int32{
	return int32(n)
}

func i64[N constraints.Integer | constraints.Float](n N) int64{
	return int64(n)
}
