#version 330

// Input vertex attributes (from vertex shader)
in vec2 fragTexCoord;
in vec4 fragColor;

// Input uniform values
uniform sampler2D texture0;

uniform sampler2D mask;
uniform sampler2D image;

uniform vec2 screenSize;
uniform vec2 imageSize;

// Output fragment color
out vec4 finalColor;

void main()
{
    // render texture is flipped so we have to invert it
    // not sure doing it in fragment shader is good idea but I don't really care
    // at this point

    vec4 maskColour = texture(mask, vec2(fragTexCoord.x, 1.0 - fragTexCoord.y));

    if (maskColour.r < 0.01){
        discard;
    }else{
        vec2 imageCoord = vec2(
            fragTexCoord.x * screenSize.x / imageSize.x,
            fragTexCoord.y * screenSize.y / imageSize.y);

        finalColor = texture(image, imageCoord);
    }
}
