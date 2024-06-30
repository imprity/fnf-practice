#version 330

// Input vertex attributes (from vertex shader)
in vec2 fragTexCoord;
in vec4 fragColor;

// Input uniform values
uniform sampler2D texture0;
uniform vec4 colDiffuse;

// Custom inputs
uniform vec4 strokeColor;
uniform vec4 fillColor;

uniform float buffer;
uniform float strokeWidth;
uniform float smoothing;

// Output fragment color
out vec4 finalColor;

void main(){
    float dist = texture(texture0, fragTexCoord).a;
    float distDelta = length(vec2(dFdx(dist), dFdy(dist)));

    float dist2 = max(texture(texture0, fragTexCoord).a, distDelta);

    float border = smoothstep(buffer + strokeWidth - smoothing, buffer + strokeWidth + smoothing, dist);
    float alpha = smoothstep(buffer - smoothing, buffer + smoothing, dist);

    finalColor = mix(strokeColor, fragColor, border) * alpha;
    //finalColor = vec4(1,1,1,dist2);
}
