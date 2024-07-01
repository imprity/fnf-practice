#version 330

// Input vertex attributes (from vertex shader)
in vec2 fragTexCoord;
in vec4 fragColor;

// Input uniform values
uniform sampler2D texture0;
uniform vec4 colDiffuse;

// Custom inputs
uniform vec4 uValues0;

// Output fragment color
out vec4 finalColor;


void main(){
    float dist1 = texture(texture0, fragTexCoord).a - uValues0.x;

    float distDelta1 = length(vec2(dFdx(dist1), dFdy(dist1)));

    float alpha = smoothstep(-distDelta1, distDelta1, dist1);

    finalColor = fragColor * alpha;
}
