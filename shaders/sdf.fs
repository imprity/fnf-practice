#version 330

// Input vertex attributes (from vertex shader)
in vec2 fragTexCoord;
in vec4 fragColor;

// Input uniform values
uniform sampler2D texture0;
uniform vec4 colDiffuse;

// Custom inputs
uniform vec4 uValues;

// Output fragment color
out vec4 finalColor;

void main(){
    float distanceFromOutline = texture(texture0, fragTexCoord).a - uValues.x + uValues.y;
    float distanceChangePerFragment = length(vec2(dFdx(distanceFromOutline), dFdy(distanceFromOutline)));
    float alpha = smoothstep(-distanceChangePerFragment, distanceChangePerFragment, distanceFromOutline);

    // Calculate final fragment color
    finalColor = fragColor * alpha;
}
