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

/*
void main(){
    float dist = texture(texture0, fragTexCoord).a;
    float distDelta = length(vec2(dFdx(dist), dFdy(dist)));

    float dist2 = max(texture(texture0, fragTexCoord).a, distDelta);

    float border = smoothstep(buffer + strokeWidth - smoothing, buffer + strokeWidth + smoothing, dist);
    float alpha = smoothstep(buffer - smoothing, buffer + smoothing, dist);

    finalColor = mix(strokeColor, fragColor, border) * alpha;
    //finalColor = vec4(1,1,1,dist2);
}
*/

void main(){
    float dist1 = texture(texture0, fragTexCoord).a - uValues.x + uValues.y;
    float dist2 = texture(texture0, fragTexCoord).a - uValues.x;

    float distDelta1 = length(vec2(dFdx(dist1), dFdy(dist1)));
    float distDelta2 = length(vec2(dFdx(dist2), dFdy(dist2)));

    float alpha1 = smoothstep(-distDelta1, distDelta1, dist1);
    float alpha2 = smoothstep(-distDelta2, distDelta2, dist2);

    // Calculate final fragment color
    finalColor = vec4(alpha1 - alpha2);
}
