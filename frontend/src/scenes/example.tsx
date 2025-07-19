import {makeScene2D, Circle} from '@motion-canvas/2d';
import {all, createRef} from '@motion-canvas/core';

export default makeScene2D(function* (view) {
  const myCircle = createRef<Circle>();
  view.fill('#000000');
  view.add(
    <Circle
      ref={myCircle}
      // Add first circle and set its properties:
      x={-300}
      width={140}
      height={140}
      fill="#e13238"
    />, 
  );

  // Add second circle (new) and set its properties:
  const secondCircle = createRef<Circle>();
  view.add(
    <Circle
      ref={secondCircle}
      x={300}
      width={140}
      height={140}
      fill="#38e132"
    />, 
  );

  yield* all(
    myCircle().position.x(300, 1).to(-300, 1),
    myCircle().fill('#e6a700', 1).to('#e13238', 1),
    secondCircle().position.x(-300, 1).to(300, 1),
    secondCircle().fill('#e13238', 1).to('#e6a700', 1),
  );
});
