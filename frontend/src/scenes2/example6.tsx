/* This example performs visualization of the interpolation function easeinoutCubic


*/import {Circle, makeScene2D} from '@motion-canvas/2d';
import {createRef, easeOutSine, map, tween, Vector2} from '@motion-canvas/core';
import { Color } from '@motion-canvas/core';
import { easeInOutCubic } from '@motion-canvas/core';
import {arcLerp} from '@motion-canvas/core';


export default makeScene2D(function* (view) {
    const circle = createRef<Circle>();
    const circle_positions: Vector2[] = []
    // 122 because frame rate times number of seconds
    let starting_x = -view.width()/4;
    let starting_y = -view.height()/4;
    for (let i = 0; i <= 122; i++) {
        const x = starting_x + i / 100 * view.width()/2;
        const y = starting_y + easeInOutCubic(i / 122) * view.height()/2;
        circle_positions.push(new Vector2(x, y))
    }
    view.add(<Circle ref={circle} x={starting_x} y={starting_y} width={100} height={100} fill="red" />)
    let iteration = 0;
  yield* tween(1.5, value => {
    circle().position(circle_positions[iteration++]);
  })})