package main

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

var renderTextureStack []rl.RenderTexture2D

func FnfBeginTextureMode(renderTexture rl.RenderTexture2D) {
	if len(renderTextureStack) > 0 {
		rl.EndTextureMode()
	}
	rl.BeginTextureMode(renderTexture)
	renderTextureStack = append(renderTextureStack, renderTexture)
}

func FnfEndTextureMode() {
	rl.EndTextureMode()

	// pop the render stack
	renderTextureStack = renderTextureStack[:len(renderTextureStack)-1]

	// if there is some thing left on render stack
	// begin texture mode
	if len(renderTextureStack) >= 1 {
		stackLast := renderTextureStack[len(renderTextureStack)-1]
		rl.BeginTextureMode(stackLast)
	}
}

/*
begin(t1)         // t1
	begin(t2)     // t1 t2
	end(t2)       // t1
	begin(t3)     // t1 t3
		begin(t4) // t1 t3 t4
		end(t4)   // t1 t3
	end(t3)       // t1
end(t1)           // t1
*/
