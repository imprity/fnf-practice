package fnf

import (
	"encoding/json"
	"fmt"
	"io"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Sprite struct {
	Texture rl.Texture2D

	Width  float32
	Height float32

	Count int

	Margin float32
}

func checkSpriteBound(sprite Sprite, spriteN int) {
	if spriteN < 0 || spriteN >= sprite.Count {
		panicMsg := fmt.Sprintf("index out of range [%d] with length %d", spriteN, sprite.Count)
		panic(panicMsg)
	}
}

func SpriteRect(sprite Sprite, spriteN int) rl.Rectangle {
	checkSpriteBound(sprite, spriteN)

	w := sprite.Width + sprite.Margin
	h := sprite.Height + sprite.Margin

	colCount := int(f32(sprite.Texture.Width) / w)
	rowCount := int(f32(sprite.Texture.Height) / h)

	_ = rowCount // might use this later

	// prevent dvidision by zero
	// (it also makes no sense for col and row count to be zero)
	colCount = max(colCount, 1)
	rowCount = max(rowCount, 1)

	col := spriteN % colCount
	row := spriteN / colCount

	return rl.Rectangle{
		Width:  sprite.Width,
		Height: sprite.Height,
		X:      f32(col) * w,
		Y:      f32(row) * h,
	}
}

func spriteSubRect(sprite Sprite, spriteN int, subRect rl.Rectangle) rl.Rectangle {
	spriteRect := SpriteRect(sprite, spriteN)

	subRect.X += spriteRect.X
	subRect.Y += spriteRect.Y

	return RectIntersect(spriteRect, subRect)
}

func DrawSprite(sprite Sprite, spriteN int, posX int32, posY int32, tint rl.Color) {
	rect := SpriteRect(sprite, spriteN)

	rl.DrawTextureRec(
		sprite.Texture, rect,
		rl.Vector2{X: f32(posX), Y: f32(posY)},
		tint,
	)
}

func DrawSpriteRec(
	sprite Sprite, spriteN int,
	sourceRec rl.Rectangle, position rl.Vector2, tint rl.Color) {

	rect := spriteSubRect(sprite, spriteN, sourceRec)

	rl.DrawTextureRec(
		sprite.Texture, rect,
		position,
		tint,
	)
}

func DrawSpriteV(sprite Sprite, spriteN int, position rl.Vector2, tint rl.Color) {
	rect := SpriteRect(sprite, spriteN)

	rl.DrawTextureRec(
		sprite.Texture, rect,
		position,
		tint,
	)
}

func DrawSpriteTransfromed(
	sprite Sprite, spriteN int,
	srcRect rl.Rectangle,
	mat rl.Matrix,
	tint rl.Color,
) {
	rect := spriteSubRect(sprite, spriteN, srcRect)

	DrawTextureTransfromed(
		sprite.Texture,
		rect, mat, tint)
}

type spriteJsonMetadata struct {
	SpriteWidth  float32
	SpriteHeight float32

	SpriteCount int

	SpriteMargin float32
}

// Parse sprite json metadata.
// Parsed sprite doen't contain texture.
func ParseSpriteJsonMetadata(jsonReader io.Reader) (Sprite, error) {
	sprite := Sprite{}
	metadata := spriteJsonMetadata{}

	decoder := json.NewDecoder(jsonReader)

	if err := decoder.Decode(&metadata); err != nil {
		return sprite, err
	}

	sprite.Width = metadata.SpriteWidth
	sprite.Height = metadata.SpriteHeight
	sprite.Margin = metadata.SpriteMargin
	sprite.Count = metadata.SpriteCount

	return sprite, nil
}
