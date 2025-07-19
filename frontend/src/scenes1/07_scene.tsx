import {makeScene2D, Rect, Node, Circle} from '@motion-canvas/2d';
import {createRef, createComputed, waitFor} from '@motion-canvas/core';

export default makeScene2D(function* (view) {
  // Refs for container and circle
  const container = createRef<Rect>();
  const ball = createRef<Circle>();

  // Dynamic radius (10% of view height)
  const radius = createComputed(() => view.height() * 0.1);
  // Bounce boundaries: top and bottom inside the container
  const topY = createComputed(() => -view.height() / 2 + radius());
  const bottomY = createComputed(() => view.height() / 2 - radius());

  // Build the scene: full-screen background and centered node
  view.add(
    <Rect
      ref={container}
      width={() => view.width()}
      height={() => view.height()}
      fill="#222"
    >
      <Node>
        <Circle
          ref={ball}
          x={() => 0}
          y={() => topY()}
          width={() => radius() * 2}
          height={() => radius() * 2}
          fill="#e13238"
        />
      </Node>
    </Rect>
  );

  // Allow one frame to render
  yield* waitFor(0);

  // Infinite bouncing loop
  while (true) {
    // Drop to bottom
    yield* ball().position.y(bottomY(), 0.8);
    // Rise to top
    yield* ball().position.y(topY(), 0.8);
  }
});