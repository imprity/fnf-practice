package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

type UpdateResult interface {
	DoQuit() bool

	QuitWithTransition() bool
	QuitTransitionTexture() rl.Texture2D
}

// common things update result needs
// purely for convenience
type UpdateResultBase struct{
	Quit bool
	TransitionTexture rl.Texture2D
}

func (u UpdateResultBase) DoQuit() bool {
	return u.Quit
}

func (u UpdateResultBase) QuitWithTransition() bool {
	return u.TransitionTexture.ID > 0
}

func (u UpdateResultBase) QuitTransitionTexture() rl.Texture2D{
	return u.TransitionTexture
}

func (u *UpdateResultBase) SetQuit(quit bool){
	u.Quit = quit
}

func (u *UpdateResultBase) SetTransitionTexture(tex rl.Texture2D){
	u.TransitionTexture = tex
}


type Screen interface {
	Update() UpdateResult
	Draw()
	BeforeScreenTransition()
}
