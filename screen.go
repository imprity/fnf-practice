package main

type UpdateResult interface{
	DoQuit() bool
}

type Screen interface{
	Update() UpdateResult
	Draw()
}
