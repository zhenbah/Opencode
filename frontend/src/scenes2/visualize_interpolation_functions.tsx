import {Circle, makeScene2D} from '@motion-canvas/2d';
import {createRef, easeOutSine, map, tween, Vector2} from '@motion-canvas/core';
import { Color } from '@motion-canvas/core';
import { easeInOutCubic } from '@motion-canvas/core';
import {arcLerp} from '@motion-canvas/core';



export default makeScene2D(function* (view) {
  const circle = createRef<Circle>();
  const circle_positions: Vector2[] = []
  for (let i = 0; i <= 100; i++) {
    const x = i / 100;
    const y = easeInOutCubic(x); // Replace with any easing function
    circle_positions.push(new Vector2(x * 500, y * 300))
  }
let iteration = 0 ;
yield*  tween(100/60,   (value) => {iteration +=1 ;circle().position(circle_positions[iteration])})
});