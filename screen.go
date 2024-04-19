package main

type Screen interface {
	Update()
	Draw()
	BeforeScreenTransition()
}
