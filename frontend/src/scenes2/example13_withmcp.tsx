
import { makeScene2D, Rect, Circle, Node, Txt } from '@motion-canvas/2d';
import { createRef, waitFor, all } from '@motion-canvas/core';

export default makeScene2D(function* (view) {
  // Create refs for the shapes
  const rectRef = createRef<Rect>();
  const circleRef = createRef<Circle>();

  // Initial positions
  const rectStart = [-200, 0];
  const circleStart = [200, 0];

  // Add rectangle and circle to the scene
  view.add(
    <>
      <Rect
        ref={rectRef}
        width={120}
        height={80}
        fill={'#3498db'}
        position={rectStart}
        radius={16}
      />
      <Circle
        ref={circleRef}
        width={80}
        height={80}
        fill={'#e74c3c'}
        position={circleStart}
      />
    </>
  );

  // Animate: swap positions
  yield* all(
    rectRef().position(circleStart, 1),
    circleRef().position(rectStart, 1),
  );
});
