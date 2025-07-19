import {Circle, makeScene2D} from '@motion-canvas/2d';
import {
  createRef, 
  easeOutSine, 
  easeInOutCubic, 
  easeInExpo,
  easeOutExpo,
  easeInOutExpo,
  linear,
  map, 
  tween, 
  Vector2
} from '@motion-canvas/core';
import { Color } from '@motion-canvas/core';

export default makeScene2D(function* (view) {

  const circle = createRef<Circle>();

  view.add(
    <Circle
      ref={circle}
      x={-300}
      width={240}
      height={240}
      fill="#e13238"
    />,
  );

  // Example of color lerp
  const colours_lerped = []
  const num_seconds_tween = 2;
  const num_iterations_tween = num_seconds_tween * 60;
  
  for (let i = 0; i < num_iterations_tween; i++) {
    colours_lerped.push(
      Color.lerp(
        new Color('red'),
        new Color('blue'), 
        i / num_iterations_tween
      )
    );
  }
  
  for (let i = 0; i < num_iterations_tween; i++) {
    colours_lerped.push(
      Color.lerp(
        new Color('blue'),
        new Color('green'), 
        i / num_iterations_tween
      )
    );
  }
  
  console.log(`len colours_lerped ${colours_lerped.length}`);
  const colours_lerped_final = colours_lerped.filter((_, index) => index % 2 === 0);
    yield*   tween(2, value => {
    const colour_1lerp =       Color.lerp(
      new Color('#e6a700'),
      new Color('#e13238'),
      easeInOutCubic(value),
    );
    console.log(`colour_1lerp ${colour_1lerp}`)
    circle().fill(colour_1lerp);
  });
});